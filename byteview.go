package DistributedCache

import "time"

/**
byteview是对实际缓存的⼀层封装，因为实际的缓存值是⼀个byte切⽚存储的，
⽽切⽚的底层是⼀个指向底层数组的指针，⼀个记录⻓度的变量和⼀个记录容量的变量。
如果获取缓存值时直接返回缓存值的切⽚，那个切⽚只是原切⽚三个变量的拷⻉，
真正的缓存值就可能被外部恶意修改。
所以⽤byteView进⾏⼀层封装，返回缓存值时的byteView则是⼀个原切⽚的深拷⻉。
*/

// ByteView 只读数据结构，使用ByteSlice返回一个拷贝
// 防止缓存值被外部程序修改
type ByteView struct {
	b []byte //  b会存储真实的缓存值
	t time.Time
	// expire time.Time//  过期时间
	// 支持多种数据结构的数据类型的存储，比如字符串、图片等
}

// 返回与此视图关联的过期时间
func (v ByteView) Expire() time.Time {
	return v.t
}

// Len return the view‘s length
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice return a copy of the data as a byte slice
// 防止缓存值被外部程序修改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// String returns the data as a string
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
