package DistributedCache

//
//import (
//	gee2 "GeeCache"
//	"flag"
//	"fmt"
//	"log"
//	"net/http"
//)
//
//var db = map[string]string{
//	"sam":    "浪子心声",
//	"lam":    "分分钟需要你",
//	"Lesile": "沉默是金",
//}
//
//func createGroup() *gee2.Group {
//	return gee2.NewGroup("scores", 2<<10, gee2.GetterFunc(
//		func(key string) ([]byte, error) {
//			log.Println("[SlowDB] search key", key)
//			if v, ok := db[key]; ok {
//				return []byte(v), nil
//			}
//			return nil, fmt.Errorf("%s not exist", key)
//		}))
//}
//
////	`startCacheServer()` 用来启动缓存服务器：创建 HTTPPool
////
//// 添加节点信息，注册到 gee 中
//// 启动 HTTP 服务（共3个端口，8001/8002/8003），用户不感知
//func startCacheServer(addr string, addrs []string, g *gee2.Group) {
//	peers := gee2.NewHTTPPool(addr)
//	peers.Set(addrs...)
//	g.RegisterPeers(peers)
//	log.Println("geecache is running at", addr)
//	log.Fatal(http.ListenAndServe(addr[7:], peers))
//}
//
//// `startAPIServer()` 用来启动一个 API 服务（端口 9999），与用户进行交互，用户感知。
//func startAPIServer(apiAddr string, g *gee2.Group) {
//	http.Handle("/api", http.HandlerFunc(
//		func(writer http.ResponseWriter, request *http.Request) {
//			key := request.URL.Query().Get("key")
//			view, err := g.Get(key)
//			if err != nil {
//				http.Error(writer, err.Error(), http.StatusInternalServerError)
//				return
//			}
//			writer.Header().Set("Content-Type", "application/octet-stream")
//			writer.Write(view.ByteSlice())
//		}))
//	log.Println("fontend server is running at", apiAddr)
//	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
//}
//
//func main() {
//	var port int
//	var api bool
//	flag.IntVar(&port, "port", 8001, "Geecache server port")
//	flag.BoolVar(&api, "api", false, "Start a api server?")
//	flag.Parse()
//
//	apiAddr := "http://localhost:9999"
//	addrMap := map[int]string{
//		8001: "http://localhost:8001",
//		8002: "http://localhost:8002",
//		8003: "http://localhost:8003",
//	}
//
//	var addrs []string
//	for _, v := range addrMap {
//		addrs = append(addrs, v)
//	}
//
//	gee := createGroup()
//	if api {
//		go startAPIServer(apiAddr, gee)
//	}
//	startCacheServer(addrMap[port], []string(addrs), gee)
//}
