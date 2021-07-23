package geecache

// A ByteView 持有一个 不可变的字节数组 b
type ByteView struct {
	b []byte
}

// Len 返回这个不可变数组的长度
func (bv ByteView) Len() int {
	return len(bv.b)
}

// String byte to string 方法
func (bv ByteView) String() string {
	return string(bv.b)
}

// ByteSlice 将字节数组拷贝成一个切片返回
func (bv ByteView) ByteSlice() []byte {
	return cloneBytes(bv.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
