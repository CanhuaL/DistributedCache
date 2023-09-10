package lru

import (
	"container/list"
	"time"
)

/**
缓存模块是数据实际存储的位置，其中实现缓存淘汰算法，
过期机制，回调机制等，缓存模块与其他部分是解耦的，
因此可以根据不同场景选择不同的缓存淘汰算法（默认lru）。
groupcache本身的实现中，缓存值只淘汰不更新，
也没有超时淘汰机制，通过这样来简化设计，并没有指定缓存的移除操作。
在本项⽬的拓展中加了指定键的移除操作，在lru中进⾏了实现。

增加了键的移除操作后，需要考虑的⼀个问题是：移除了某个键，每个节点中的热点缓存中如果有该键，
也需要删除，因此删除操作需要通知所有节点（etcd实现）。
另外对于缓存来说，超时淘汰有时候也是必要的，因此在缓存中对于每个value，都设计了过期时间。
*/
// Cache 是LRU缓存。并发访问是不安全的

type NowFunc func() time.Time

type Cache struct {
	Now      NowFunc
	maxBytes int64                    // 允许使用的最大内存，超过该大小会采用淘汰策略
	nBytes   int64                    // 当前已经使用的内存大小
	ll       *list.List               // 双向链表存储缓存数据
	cache    map[string]*list.Element // 字典，值是双向链表中对应节点的指针
	// 可选，清理条目时调用
	onEvicted func(key string, value Value) // 某条记录被移除时的回调函数，可以为 nil
}

//	键值对entry，双向链表节点的数据类型，在链表中仍需要保存每个值对应的key
//
// 淘汰队尾节点时，需要key从字典中删除对应的映射
type entry struct {
	key    string
	value  Value
	expire time.Time
}

// 为了通用性，我们允许值实现了Value接口的任意类型
// 该接口只包含了一个方法Len（）int，用于返回值所占用的内存大小
type Value interface {
	Len() int
}

// 实例化Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes: maxBytes,
		// nBytes:    0,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: onEvicted,
		Now:       time.Now,
	}
}

/*
*
Get方法，通过key在cache中寻找数据，如果查找到就将节点移动到对头
并且返回数据
*/
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// 已存在，节点移动到队头。（队尾的节点会被优先淘汰）
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		//  如果条目已过期，请将其从缓存中删除
		if !kv.expire.IsZero() && kv.expire.Before(c.Now()) {
			c.removeElement(ele)
			return nil, false
		}
		return kv.value, true
	}
	return
}

// 删除一个节点
func (c *Cache) removeElement(e *list.Element) {
	c.ll.Remove(e)
	kv := e.Value.(*entry)
	delete(c.cache, kv.key)
	if c.onEvicted != nil {
		c.onEvicted(kv.key, kv.value)
	}
}

func (c *Cache) RemoveOldest() {
	// 取队首节点删除
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		// 	回调函数
		if c.onEvicted != nil {
			c.onEvicted(kv.key, kv.value)
		}
	}
}

/*
*
Add，增加结点方法，先通过key查询，如果key存在直接将结点移至队伍头，
不存在则创建结点&entry
*/
func (c *Cache) Add(key string, val Value, expir time.Time) {
	if ele, ok := c.cache[key]; ok {
		// 已存在，节点移动到队尾。（队首的节点会被优先淘汰）
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		// 已用内存，此处计算新增的内存
		c.nBytes += int64(val.Len()) - int64(kv.value.Len())
		kv.value = val
		kv.expire = expir
	} else {
		ele := c.ll.PushFront(&entry{
			key:    key,
			value:  val,
			expire: expir,
		})
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(val.Len())
	}
	// 判断是否超过设定的最大值maxBytes
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.RemoveOldest()
	}
}

// Len Value接口实现
func (c *Cache) Len() int {
	return c.ll.Len()
}
