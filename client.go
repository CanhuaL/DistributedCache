package GeeCache

import (
	pb "GeeCache/geecachepb"
	"GeeCache/registry"
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

// client 模块实现geecache访问其他远程节点 从而获取缓存的能力

type client struct {
	name string //  服务名称 pcache/ip:addr
}

// Fetch  从remote peer获取对应缓存值
func (c *client) Fetch(group string, key string) ([]byte, error) {
	//  创建一个etcd client
	cli, err := clientv3.New(defaultEtcdConfig)
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	//  发现服务  取得与服务的连接
	conn, err := registry.EtcdDial(cli, c.name)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	grpcClient := pb.NewGroupCacheClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := grpcClient.Get(ctx, &pb.Request{
		Group: group,
		Key:   key,
	})
	if err != nil {
		return nil, fmt.Errorf("could not get %s/%s from peer %s", group, key, c.name)
	}
	return resp.GetValue(), nil
}

func NewClient(service string) *client {
	return &client{name: service}
}
