package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

//分布式⼀致性模块实现了⼀致性哈希算法，将机器节点组成哈希环，
//为每个节点提供了从其他节点获取缓存的能⼒

// 定义了函数类型 `Hash`，采取依赖注入的方式，允许用于替换成自定义的 Hash 函数
type Hash func(data []byte) uint32

// 哈希算法的主要结构体
type Map struct {
	hash     Hash           //  hash函数
	replicas int            //  虚拟节点倍数
	keys     []int          //  哈希环
	hashMap  map[int]string //  虚拟节点与真实节点的映射表
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}

	return m
}

/*
*
`Add` 函数允许传入 0 或 多个真实节点的名称。
对每一个真实节点 `key`，对应创建 `m.replicas` 个虚拟节点，
虚拟节点的名称是：`strconv.Itoa(i) + key`，即通过添加编号的方式区分不同虚拟节点。
使用 `m.hash()` 计算虚拟节点的哈希值，使用 `append(m.keys, hash)` 添加到环上。
在 `hashMap` 中增加虚拟节点和真实节点的映射关系。
最后一步，环上的哈希值排序。
*/
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			//  用byte切片来接收key[108 105 97 110 103]
			//  再通过hash函数转成数字，起到一个临时接收的变量的效果
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

/*
*
计算 key 的哈希值
顺时针找到第一个匹配的虚拟节点的下标idx，
从m.keys中获取到对应的哈希值。
如果idx == len(m.keys)，说明应选择m.keys[0]，
因为m.keys是一个环状结构，所以用取余数的方式来处理这种情况
通过hashMap映射得到真实的节点
*/
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	//  idx虚拟结点的下标
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]]
}
