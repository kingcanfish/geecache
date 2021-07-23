package geecache

import "sync"

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
