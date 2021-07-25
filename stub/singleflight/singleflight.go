package singleflight

import "sync"

// call 代表进行中 或者已经结束的请求
// 使用 sync.WaitGroup 避免锁重入
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group 是SingleFlight 的主结构
// 管理不同key 的 call
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		//如果有Call ,说明之前已经进行了查询请求
		//释放锁并等待结果进行返回
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	// 没有的话 创建请求并且等待
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
