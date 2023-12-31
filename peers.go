package DistributedCache

//抽象出 2 个接口，PeerPicker 的 `PickPeer()` 方法用于根据传入的 key 选择相应节点 PeerGetter
//接口 PeerGetter 的 `Get()` 方法用于从对应 group 查找缓存值。PeerGetter 就对应于上述流程中的 HTTP 客户端

type PeerPicker interface {
	PickPeer(key string) (Fetcher, bool)
}

type Fetcher interface {
	//Get(group string, key string) ([]byte, error)
	Fetch(group string, key string) ([]byte, error)
}
