package DistributedCache

import (
	pb "DistributedCache/geecachepb"
	"DistributedCache/singleflight"
	"fmt"
	"log"
	"sync"
	"time"
)

/*
*Group 是 GeeCache 最核心的数据结构
负责与用户的交互，并且控制缓存值存储和获取的流程
*/
//  要求对象实现从数据源获取数据的能力
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 定义函类型并实现了Getter的接口方法
type GetterFunc func(key string) ([]byte, error)

// Get 实现Getter接口方法，使得任意匿名函数func
// GetterFunc(func)类型强制转换后，实现了 Get 接口的能力
func (f GetterFunc) Get(key string) ([]byte, error) {
	// 在方法内调用自己，可以将其它函数（GetterFunc）转换为接口（Getter）
	return f(key)
}

// Group模块是对外提供服务接⼝的部分，⼀个Group就是⼀个缓存空间。
// 其要实现对缓存的增删查⽅法。
type Group struct {
	name      string //  缓存空间的名字
	getter    Getter //	数据源获取数据
	mainCache cache  //	主缓存，并发缓存
	//hotCache  cache//热点缓存
	peers  PeerPicker          //	用于获取远程节点请求客户端
	loader *singleflight.Group //	避免对同一个key多次加载造成缓存击穿
	//emptyKeyDuration time.Duration//  getter返回error时对应空值key的过期时间
}

var (
	mu     sync.RWMutex //	读写锁
	groups = make(map[string]*Group)
)

/**
  NewGroup 实例化Group
  一个Group可以认为是一个缓存空间
每个 Group 拥有一个唯一的名称 `name`
比如可以创建三个 Group，缓存学生的成绩命名为 scores
缓存学生信息的命名为 info，缓存学生课程的命名为 courses
*/

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,                        //  缓存未命中时，获取源数据的回调函数（callback）
		mainCache: cache{cacheBytes: cacheBytes}, //  一开始实现的并发缓存
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group created with NewGroup or nil if groups not exist
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

//// RegisterSvr 为 Group 注册 peers
//func (g *Group) RegisterSvr(p PeerPicker) {
//	if g.peers != nil {
//		panic("group had been registered server")
//	}
//	g.peers = p
//}

// 删除groups映射
func DestroyGroup(name string) {
	g := GetGroup(name)
	if g != nil {
		svr := g.peers.(*server)
		svr.Stop()
		delete(groups, name)
		log.Printf("Destory cache [%s %s]", name, svr.addr)
	}
}

// 命中缓存就返回，不然就调用load去获取
func (g *Group) Get(key string, expir time.Time) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	return g.load(key, expir)
}

// getLocally调用用户回调函数g.getter.Get(key)获取数据
//
//	本地向Retriever取回数据并填充缓存
func (g *Group) getLocally(key string, expir time.Time) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value, expir)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView, expir time.Time) {
	g.mainCache.add(key, value, expir)
}

// `RegisterPeers()` 方法，将 实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中。
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 修改 load 方法，使用 `PickPeer()` 方法选择节点，若非本机节点
// 则调用 `getFromPeer()` 从远程获取。若是本机节点或失败，则回退到 `getLocally()`
// 使用 `g.loader.Do` 包裹起来即可，这样确保了并发场景下针对相同的 key，`load` 过程只会调用一次。
func (g *Group) load(key string, expir time.Time) (value ByteView, err error) {
	//若非本机节点则调用 `getFromPeer()`
	view, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			//if peer, ok := g.peers.PickPeer(key); ok {
			//	if value, err = g.getFromPeer(peer, key); err == nil {
			//		return value, nil
			//	}
			//	log.Println("[GeeCaChe] Failed to get from peer", err)
			//}
			if fetcher, ok := g.peers.PickPeer(key); ok {
				bytes, err := fetcher.Fetch(g.name, key)
				if err == nil {
					return ByteView{b: cloneBytes(bytes)}, nil
				}
				log.Printf("fail to get *%s* from peer, %s.\n", key, err.Error())
			}
		}
		return g.getLocally(key, expir)
	})
	if err == nil {
		return view.(ByteView), nil
	}
	return ByteView{}, err
}

// `getFromPeer()` 方法，使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer Fetcher, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	_, err := peer.Fetch(req.Group, req.Key)
	//bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
