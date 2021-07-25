package geecache

// PeerPicker 的 PickPeer() 方法用于根据传入的 key 选择相应节点
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

//PeerGetter 的 Get() 方法用于从对应 group 查找缓存值
//PeerGetter 就对应 HTTP 客户端 HTTP客户端实现了这个借口
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
