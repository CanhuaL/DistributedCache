package DistributedCache

import (
	"GeeCache/lru"
	"sync"
	"time"
)

/*
*
实例化lru，封装get和add方法，并添加互斥锁mu
*/
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

// 新增缓存，加锁支持并发安全
func (c *cache) add(key string, value ByteView, expir time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 延迟加载
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value, expir)
}

// 获取缓存
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok //  类型断言
	}
	return
}
