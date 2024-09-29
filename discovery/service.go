package discovery

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"	
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/m5m/model"
)

type DiscoveryService struct {
	redisClient *redis.Client
}

func NewDiscoveryService(rdb *redis.Client) *DiscoveryService {
    return &DiscoveryService{
        redisClient: rdb,
    }
}


func (d *DiscoveryService) Discover(containerName string) model.Container {

	ctx := context.Background()
    id, err := d.redisClient.Get(ctx, "id." + containerName).Result()
    if err != nil {
        log.Printf("Error retrieving value from Redis: %v", err)
        return model.Container{}
    }
	if id == "" {
		log.Printf("Container %s not found", containerName)
		return model.Container{}
	}
	return d.getContainerById(id)

}

func (d *DiscoveryService) AllocatePort(pod model.Pod) string {
	ctx := context.Background()	
	rand.NewSource(time.Now().UnixNano())
	minPort := 49152
    maxPort := 65535
    randomPortInRange := rand.Intn(maxPort-minPort+1) + minPort
	_, err := d.redisClient.SetNX(ctx, "port." + pod.Name, randomPortInRange, 1).Result()
	if err != nil {
		log.Printf("Failed Allocate Port: %v", err)
		return "9191"
	}
	return strconv.Itoa(randomPortInRange)	
}

func (d *DiscoveryService) Register(container model.Container) {
	ctx := context.Background()
	jsonContainer, err := json.Marshal(container)
	log.Default().Println("register" + string(jsonContainer))
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}
	_, err = d.redisClient.Set(ctx, "id." + container.Name, container.ContainerId, 0).Result()
	if err != nil {
		log.Printf("Error setting value in Redis: %v", err)
		return
	}
	_, err = d.redisClient.Set(ctx, "json." + container.ContainerId, jsonContainer, 0).Result()
	if err != nil {
		log.Printf("Error setting value in Redis: %v", err)
		return
	}
	d.redisClient.SAdd(ctx, "c." + container.Pod, container.ContainerId)
}

func (d *DiscoveryService) GetContainersByPod(podName string) []model.Container {
	ctx := context.Background()	
	containers := make([]model.Container, 0)
	list := d.redisClient.SMembers(ctx, "c." + podName).Val();	
	for i := 0; i < len(list); i++ {
		c := d.getContainerById(list[i])
		containers = append(containers, c)
	}
	return containers
}

func (d *DiscoveryService) Unregister(container model.Container) {
	ctx := context.Background()
	log.Println("Unregistering container " + container.ContainerId)
	_, err := d.redisClient.Del(ctx, "id." + container.Name).Result()
	if err != nil {
		log.Printf("Error deleting value from Redis: %v", err)
		return
	}
	_, err = d.redisClient.Del(ctx, "json." + container.ContainerId).Result()
	if err != nil {
		log.Printf("Error deleting value from Redis: %v", err)
		return
	}
	d.redisClient.SRem(ctx, "c." + container.Pod, container.ContainerId)
}

func (d *DiscoveryService) getContainerById(containerId string) model.Container {
	ctx := context.Background()
	jsonContainer, err := d.redisClient.Get(ctx, "json." + containerId).Result()
	if (err != nil) {
		log.Printf("Error retrieving value from Redis: %v", err)
		return model.Container{}
	}
	var container model.Container
	err = json.Unmarshal([]byte(jsonContainer), &container)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		return model.Container{}
	}	
	return container
}

func (d *DiscoveryService) RegisterPod(pod model.Pod) {
	ctx := context.Background()
	jsonPod, _ := json.Marshal(pod)
	_, err := d.redisClient.Set(ctx, "pod." + pod.Name, jsonPod, 0).Result()
	if err != nil {
		log.Printf("Error setting value in Redis: %v", err)
		return
	}
}

func (d *DiscoveryService) UnregisterPod(podName string) {
	ctx := context.Background()
	_, err := d.redisClient.Del(ctx, "pod." + podName).Result()
	if err != nil {
		log.Printf("Error deleting value from Redis: %v", err)
		return
	}
}

func (d *DiscoveryService) GetPod(podName string) model.Pod {
	ctx := context.Background()
	jsonPod, err := d.redisClient.Get(ctx, "pod." + podName).Result()
	if err != nil {
		log.Printf("Error retrieving value from Redis: %v", err)
		return model.Pod{}
	}
	var pod model.Pod
	err = json.Unmarshal([]byte(jsonPod), &pod)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		return model.Pod{}
	}
	return pod
}
