package GeeCache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

var db1 = map[string]string{
	"Leslie": "沉默是金",
	"Lam":    "分分钟需要你",
	"Sam":    "浪子心声",
}

func TestGet(t *testing.T) {
	// 统计某个键调用回调函数的次数，如果次数大于1，则表示调用了多次回调函数，没有缓存。
	loadCounts := make(map[string]int, len(db1))
	gee := NewGroup("mussic", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			//  缓存不存在  回调函数到db中查找
			log.Println("[SlowDB] search key", key)
			if v, ok := db1[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	//---------------------上面是回调函数------------------------------------------
	for k, v := range db1 {
		if view, err := gee.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		} //  load from callback function
		// 统计某个键调用回调函数的次数，如果次数大于1，则表示调用了多次回调函数，没有缓存。
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} //  cache hit
	}

	if view, err := gee.Get("unknow"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}
