package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

// Map 包含了所有的 hash key
type Map struct {
	hash     Hash           // Hash 函数
	replicas int            //虚拟节点倍数
	keys     []int          // 哈希环 keys Sorted
	hashMap  map[int]string // 真实节点和虚拟节点的映射标 key 是虚拟节点 value 是真实节点
}

// New 用来生成一个一致性 hash 实例
func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		keys:     nil,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		//生成多个虚拟副本
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get 方法通过传入的 key 找到真实节点
func (m *Map) Get(key string) string {
	if len(key) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	// 因为一致性哈希是环状结构
	// 所以用取余数来实现环状结构
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
