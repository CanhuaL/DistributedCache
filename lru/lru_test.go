package lru

import (
	"reflect"
	"testing"
	"time"
)

var expir time.Time

type String string

// Value是一个接口类型，实现了Len（）方法
// 只要类型实现了Len（）方法就是继承了Value
func (d String) Len() int {
	return len(d)
}

/*
*
实例化cache，在cache中添加数据，再查询，能通过则证明没问题
*/
func TestGet(t *testing.T) {
	lru := New(int64(0), nil)
	lru.Add("key1", String("1234"), expir)
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

/*
测试当cache数据溢出时会不会自动删除
*/
func TestRemoveOldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1 + k2 + v1 + v2)
	lru := New(int64(cap), nil)
	lru.Add(k1, String(v1), expir)
	lru.Add(k2, String(v2), expir)
	lru.Add(k3, String(v3), expir)
	//  如果能查询到key1则证明出现问题
	if _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatalf("Removeoldest key1 failed")
	}
}

/*
*
测试回调函数是否被调用
*/
func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	lru := New(int64(1), callback)
	lru.Add("key1", String("k1"), expir)
	lru.Add("key2", String("k2"), expir)
	lru.Add("key3", String("k3"), expir)
	lru.Add("key4", String("k4"), expir)

	expect := []string{"key3", "key4"}

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", keys)
	}
}

// 测试add方法
func TestAdd(t *testing.T) {
	lru := New(int64(1000), nil)
	lru.Add("key", String("1"), expir)
	lru.Add("key", String("111"), expir)

	if lru.nBytes != int64(len("key")+len("111")) {
		t.Fatal("expected 6 but got", lru.nBytes)
	}
}
