package geecache

import (
	"fmt"
	"github.com/kingcanfish/geecache/singleflight"
	"log"
	"sync"
)

// A Getter loads data for a key.
type Getter interface {
	Get(key string) ([]byte, error)
}

// A GetterFunc implements Getter with a function.
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group 可以认为是一个缓存的命名空间
// 每个 Group 拥有一个唯一的名称 name
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker

	// 使用 singleflight.Group 来确保每次远端缓存查询只进行一次
	loader *singleflight.Group
}

var (
	mu     = sync.RWMutex{}
	groups = make(map[string]*Group)
)

// NewGroup 方法创建了一个新的 Group 实例
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:   name,
		getter: getter,
		mainCache: cache{
			mu:         sync.Mutex{},
			lru:        nil,
			cacheBytes: cacheBytes,
		},
		loader: &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup 通过group name 返回所需要的 Group 实例
// 如果没有的话返回 nil
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// Get 方法为从该 Group main cache 获取缓存值
// 如果不存在则调用 load 方法, load 调用 getLocally
//（分布式场景下会调用 getFromPeer 从其他节点获取
// getLocally 调用用户回调函数 g.getter.Get() 获取源数据
// 并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.mainCache.get(key); ok {
		return v, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	// each key is only fetched once (either locally or remotely)
	// regardless of the number of concurrent callers.
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}

		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getLocally(key string) (ByteView, error) {
	// 回调函数 从用户传入的 func 中获取
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}
