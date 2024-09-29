package scheduler

import (
	"context"
	

	"github.com/go-redis/redis/v8"
	"github.com/m5m/model"
)


type Nodes struct {	
	redisClient *redis.Client
}

func NewNodes(redisClient *redis.Client) *Nodes {
	list := redisClient.SMembers(context.Background(), "nodes").Val()	
	if len(list) == 0 {
		redisClient.SAdd(context.Background(), "nodes", "tcp://localhost:2375")
	}
	return &Nodes{	
		redisClient: redisClient,	
	}
}

func (n *Nodes) PickHost(c model.Pod) string {
	hosts := n.GetHosts()
	if len(hosts) == 0 {
		return "tcp://localhost:2375"
	}
	return n.GetHosts()[0]
}

func (n *Nodes) GetHosts() []string {	
	list := n.redisClient.SMembers(context.Background(), "nodes").Val()		
	return list
}

func (n *Nodes) Register(host string) {	
	n.redisClient.SAdd(context.Background(), "nodes", host)
}

func (n *Nodes) Unregister(host string) {
	n.redisClient.SRem(context.Background(), "nodes", host)
}