package lru

import "container/list"

// Cache 是一个LRU实现的 Cache 是并发不安全的
type Cache struct {
	maxBytes  int64                         //maxBytes 是允许使用的最大内存
	nBytes    int64                         // nBytes 当前已经使用的内存
	ll        *list.List                    //双向链表实例
	cache     map[string]*list.Element      // 使用哈希表加快查找
	onEvicted func(key string, value Value) //可选的 用来清楚条目时执行(当某个条目被清除时的回调函数)
}

// entry 双向链表节点的结构类型
type entry struct {
	key   string
	value Value
}

// Value 通过 Len()方法来获取它占了多少字节
type Value interface {
	Len() int
}

// New 函数用来创建 Cache 实例
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: onEvicted,
	}
}

// Get 方法获取节点
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoveOldest 移除最近最少访问节点 即链表最尾端的元素
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.onEvicted != nil {
			c.onEvicted(kv.key, kv.value)
		}
	}
}

// Add 添加或者修改元素 同时如果超出最大容量 线性移除最后一个元素
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele = c.ll.PushFront(&entry{
			key:   key,
			value: value,
		})
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.RemoveOldest()
	}
}

// Len 链表长度方法的捷径 获取一共多少个方法
func (c *Cache) Len() int {
	return c.ll.Len()
}
