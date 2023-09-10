package GeeCache

import (
	"GeeCache/consistenthash"
	pb "GeeCache/geecachepb"
	"GeeCache/registry"
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

/**
server模块为geecache之间提供通信能力
这样部署在其他机器上的cache可以通过访问server获取缓存
至于找哪台主机 那就是一致性哈希的工作了
*/

const (
	defaultAddr = "127.0.0.1:6324"
	//defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

var (
	defaultEtcdConfig = clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}
)

// server 和 Group 是解耦合的 所以server要自己实现并发控制
type server struct {
	pb.UnimplementedGroupCacheServer

	addr       string     // format: ip:port
	status     bool       // true: running false: stop
	stopSignal chan error // 通知registry revoke服务

	//self     string //  记录自己的地址，IP和端口
	//basePath string //  作为节点间通讯地址的前缀，默认是/_geecache/
	mu    sync.Mutex
	peers *consistenthash.Map //  一致性哈希算法的Map，根据具体的key选择节点
	//  映射远程节点与对应的 httpGetter。每一个远程节点对应一个 httpGetter，因为 httpGetter 与远程节点的地址 `baseURL` 有关
	clients map[string]*client
}

// NewServer 创建cache的svr 若addr为空 则使用defaultAddr
func NewServer(addr string) (*server, error) {
	//return &HTTPPool{
	//	self:     self,
	//	basePath: defaultBasePath,
	//}
	if addr == "" {
		addr = defaultAddr
	}
	if !validPeerAddr(addr) {
		return nil, fmt.Errorf("invalid addr %s, it should be x.x.x.x:port", addr)
	}
	return &server{addr: addr}, nil
}

// Log info with server name
//func (h *HTTPPool) Log(format string, v ...interface{}) {
//	log.Println("[Server %s] %s", h.self, fmt.Sprintf(format, v...))
//}

// Start  启动cache服务
func (h *server) Start() error {
	h.mu.Lock()
	// 1. 设置status为true 表示服务器已在运行
	if h.status == true {
		h.mu.Unlock()
		return fmt.Errorf("server already started")
	}
	// -----------------启动服务----------------------
	// 1. 设置status为true 表示服务器已在运行
	// 2. 初始化stop channal,这用于通知registry stop keep alive
	// 3. 初始化tcp socket并开始监听
	// 4. 注册rpc服务至grpc 这样grpc收到request可以分发给server处理
	// 5. 将自己的服务名/Host地址注册至etcd 这样client可以通过etcd
	//    获取服务Host地址 从而进行通信。这样的好处是client只需知道服务名
	//    以及etcd的Host即可获取对应服务IP 无需写死至client代码中
	// ----------------------------------------------
	// 2. 初始化stop channal,这用于通知registry stop keep alive
	h.status = true
	h.stopSignal = make(chan error)

	port := strings.Split(h.addr, ":")[1]
	// 3. 初始化tcp socket并开始监听
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	// 4. 注册rpc服务至grpc 这样grpc收到request可以分发给server处理
	grpcServer := grpc.NewServer()
	// 5. 将自己的服务名/Host地址注册至etcd 这样client可以通过etcd
	//    获取服务Host地址 从而进行通信。这样的好处是client只需知道服务名
	//    以及etcd的Host即可获取对应服务IP 无需写死至client代码中
	pb.RegisterGroupCacheServer(grpcServer, h)

	//  注册服务到etcd
	go func() {
		// Register never return unless stop singnal received
		err := registry.Register("geecache", h.addr, h.stopSignal)
		if err != nil {
			log.Fatalf(err.Error())
		}
		//  Close channel
		close(h.stopSignal)
		//  Close tcp listen
		err = lis.Close()
		if err != nil {
			log.Fatalf(err.Error())
		}
		log.Printf("[%s] Revoke service and close tcp socket ok.", h.addr)
	}()

	h.mu.Unlock()
	if err := grpcServer.Serve(lis); h.status && err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}
	return nil
}

/**
GeeCache服务端，客户端传来数据，通过传来的groupname和key
到缓存中查询，命中再返回给前端
*/
// ServeHTTP handle all http requests
//func (h *HTTPPool) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
//	//首先判断访问路径的前缀是否是 `basePath`，不是返回错误
//	if !strings.HasPrefix(request.URL.Path, h.basePath) {
//		panic("HTTPPool serving unexpected path:" + request.URL.Path)
//	}
//
//	h.Log("%s %s", request.Method, request.URL.Path)
//	//访问路径格式为 `/<basepath>/<groupname>/<key>`
//	parts := strings.SplitN(request.URL.Path[len(h.basePath):], "/", 2)
//	if len(parts) != 2 {
//		http.Error(writer, "bad request", http.StatusBadRequest)
//		return
//	}
//
//	groupName := parts[0]
//	key := parts[1]
//
//	group := GetGroup(groupName)
//
//	if group == nil {
//		http.Error(writer, "no such group:"+groupName, http.StatusNotFound)
//		return
//	}
//
//	view, err := group.Get(key)
//	//  ServeHTTP()中使用 proto.Marshal()编码 HTTP 响应
//	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
//
//	if err != nil {
//		http.Error(writer, err.Error(), http.StatusInternalServerError)
//		return
//	}
//	if err != nil {
//		http.Error(writer, err.Error(), http.StatusInternalServerError)
//	}
//
//	writer.Header().Set("Content-Type", "application/octet-stream")
//	//使用 `w.Write()` 将缓存值作为 httpResponse 的 body 返回。
//	writer.Write(body)
//}

// baseURL 表示将要访问的远程节点的地址，例如http://example.com/_geecache/
//type httpGetter struct {
//	baseURL string
//}

// 使用 http.Get()方式获取返回值，并转换为 []bytes 类型
// Get() 中使用 proto.Unmarshal()解码 HTTP 响应
// Get 实现geeCache service的Get接口
func (h *server) Get(ctx context.Context, in *pb.Request) (*pb.Response, error) {
	//u := fmt.Sprintf(
	//	"%v%v/%v",
	//	h.baseURL,
	//	url.QueryEscape(in.GetGroup()),
	//	url.QueryEscape(in.GetKey()),
	//)
	//res, err := http.Get(u)
	//if err != nil {
	//	return err
	//}
	//defer res.Body.Close()
	//
	//if res.StatusCode != http.StatusOK {
	//	return fmt.Errorf("server returned:%v", res.Status)
	//}
	//
	//bytes, err := ioutil.ReadAll(res.Body)
	////Get() 中使用 proto.Unmarshal()解码 HTTP 响应
	//if err = proto.Unmarshal(bytes, out); err != nil {
	//	return fmt.Errorf("reading response body: %v", err)
	//}
	//
	//return nil
	//-------------------以上使用 proto.Unmarshal()解码 HTTP 响应
	group, key := in.GetGroup(), in.GetKey()
	resp := &pb.Response{}

	log.Printf("[peanutcache_svr %s] Recv RPC Request - (%s)/(%s)", h.addr, group, key)
	if key == "" {
		return resp, fmt.Errorf("key required")
	}

	g := GetGroup(group)
	if g == nil {
		return resp, fmt.Errorf("group not found")
	}
	expir := time.Time{}
	view, err := g.Get(key, expir)
	if err != nil {
		return resp, err
	}
	resp.Value = view.ByteSlice()
	return resp, nil
}

// Set 将各个远端主机IP配置到HTTPPool里
// 这样HTTPPool就可以Pick他们了
// 注意: 此操作是*覆写*操作！
// 注意: peersIP必须满足 x.x.x.x:port的格式
// set方法实例化了一致性哈希算法，并且添加了传入的节点
func (h *server) Set(peers ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	//  为每一个节点创建了一个HTTP客户端httpGetter
	h.peers = consistenthash.New(defaultReplicas, nil)
	//  peers一致性哈希算法的key
	//h.peers.Add(peers...)
	//h.httpGetter = make(map[string]*httpGetter, len(peers))
	//for _, peer := range peers {
	//	h.httpGetter[peer] = &httpGetter{baseURL: peer + h.basePath}
	//}
	h.peers.Add(peers...)
	h.clients = make(map[string]*client)
	for _, peerAddr := range peers {
		if !validPeerAddr(peerAddr) {
			panic(fmt.Sprintf("[peer %s] invalid address format, it should be x.x.x.x:port", peerAddr))
		}
		service := fmt.Sprintf("geecache/%s", peerAddr)
		h.clients[peerAddr] = NewClient(service)
	}
}

//	 根据一致性哈希选举出key应存放在的cache
//	 return false代表从本地获取cache
//		PickerPeer()` 包装了一致性哈希算法的 `Get()` 方法
//		根据具体的 key，选择节点，返回节点对应的 HTTP 客户端。
//		实现peerPicker接口
func (h *server) PickPeer(key string) (Fetcher, bool) {
	h.mu.Lock()
	defer h.mu.Lock()
	//if peer := h.peers.Get(key); peer != "" && peer != h.self {
	//	h.Log("Pick peer %s", peer)
	//	return h.httpGetter[peer], true
	//}
	//return nil, false
	peerAddr := h.peers.Get(key)
	if peerAddr == h.addr {
		log.Printf("ooh! pick myself, I am %s\n", h.addr)
		return nil, false
	}
	log.Printf("[cache %s] pick remote peer: %s\n", h.addr, peerAddr)
	return h.clients[peerAddr], true
}

// Stop停止server运行 如果server没有运行 这将是一个no-op
func (h *server) Stop() {
	h.mu.Lock()
	if h.status == false {
		h.mu.Unlock()
		return
	}
	h.stopSignal <- nil //  发送停止keepalive信号
	h.status = false    //  设置server运行状态为stop
	h.clients = nil     //  清空一致性哈希信息 有助于垃圾回收
	h.peers = nil
	h.mu.Unlock()
}
