package singleflight

import "sync"

// singlefilght 为GeeCache提供缓存击穿的保护
// 当cache并发访问peer获取缓存时 如果peer未缓存该值
// 则会向db发送大量的请求获取 造成db的压力骤增
// 因此 将所有由key产生的请求抽象成Group
// 这个Group只会起飞一次(single) 这样就可以缓解击穿的可能性
// Group载有我们要的缓存数据 称为call

// call 代表正在进行中，或已经结束的请求。使用 `sync.WaitGroup` 锁避免重入
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// singleflight 的主数据结构，管理不同 key 的请求(call)
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

/*
Do 方法，接收 2 个参数，第一个参数是 `key`，
第二个参数是一个函数 `fn`。Do 的作用就是，
针对相同的 key，无论 Do 被调用多少次，函数 `fn` 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。
使用闭包给Do方法调用,只有fn返回后，Do方法才会返回。
并发协程之间不需要消息传递，非常适合 `sync.WaitGroup`。
- wg.Add(1) 锁加1。
- wg.Wait() 阻塞，直到锁被释放。
- wg.Done() 锁减1。
*/
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	//  第一个get(key)请求到来时，如果call存在则返回数据
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()         //后续的请求只需要等待第一个请求处理完成
		return c.val, c.err //  请求结束返回结果
	}
	//  不存在,则去创建
	c := new(call)
	c.wg.Add(1)  //  发起请求前加锁
	g.m[key] = c //  添加到g.m 表明唯一的call的key已经有对应的请求在处理
	g.mu.Unlock()

	c.val, c.err = fn() //  调用fn发起请求，执行传入的函数，将结果存储到call结构体的字段中
	c.wg.Done()         //  请求结束

	g.mu.Lock()
	delete(g.m, key) //  已完成,更新g.m
	g.mu.Unlock()

	return c.val, c.err //  返回结果
}
