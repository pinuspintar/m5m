package api

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/m5m/discovery"
	"github.com/m5m/model"
	"github.com/m5m/provider/docker"
	"github.com/m5m/scheduler"
)

type ApiService struct {
	ds    *discovery.DiscoveryService
	nodes *scheduler.Nodes
}

func ApiServiceNew(ds *discovery.DiscoveryService, nodes *scheduler.Nodes) *ApiService {
	return &ApiService{
		ds:    ds,
		nodes: nodes,
	}
}

func (a *ApiService) GetNodes() []string {
	return a.nodes.GetHosts()
}

func (a *ApiService) GetAllContainers() []model.Container {
	containers := make([]model.Container, 0)
	hosts := a.nodes.GetHosts()
	if len(hosts) == 0 {
		hosts = []string{"tcp://localhost:2375"}
	}
	for _, host := range hosts {
		dockerClient := docker.DockerServiceNew(host)
		cs, _ := dockerClient.ListContainers()
		for _, c := range cs {
			a.ds.Register(c)
		}
		containers = append(containers, cs...)
		defer dockerClient.Close()
	}
	return containers
}

func (a *ApiService) GetContainersByPod(podName string) []model.Container {
	list := a.ds.GetContainersByPod(podName)
	if len(list) == 0 {
		l := a.GetAllContainers()
		if len(l) == 0 {
			log.Println("No containers found for pod " + podName)
			return nil
		}
	}
	containers := make([]model.Container, 0)
	for _, c := range list {
		dockerClient := docker.DockerServiceNew(c.Host)
		cJson, err := dockerClient.Inspect(c.ContainerId)
		if err != nil && strings.Contains(err.Error(), "No such container") {
			a.ds.Unregister(c)
			continue
		}
		log.Println(cJson.Config.Labels)
		containers = append(containers, model.Container{
			ContainerId:   cJson.ID,
			Pod:           podName,
			Image:         cJson.Image,
			Port:          cJson.Config.Labels["port"],
			ContainerPort: cJson.Config.Labels["containerPort"],
			Name:          cJson.Name,
			Status:        cJson.State.Status,
			Command:       strings.Join(cJson.Config.Cmd, "\n"),
			Host:          c.Host,
			Age:           time.Since(time.Unix(c.CreatedAt, 0)).String(),
			CreatedAt:     c.CreatedAt,
		})
		defer dockerClient.Close()
	}
	return containers
}

func (a *ApiService) Inspect(containerName string) types.ContainerJSON {
	c := a.ds.Discover(containerName)
	dockerClient := docker.DockerServiceNew(c.Host)
	defer dockerClient.Close()
	i, _ := dockerClient.Inspect(c.ContainerId)
	return i
}

func (a *ApiService) Apply(pod model.Pod) (model.Container, error) {
	a.ds.RegisterPod(pod)
	
	exposedPort := a.ds.AllocatePort(pod)
	
	log.Println(fmt.Printf("Apply pod %v", pod))
	host := a.nodes.PickHost(pod)
	dockerClient := docker.DockerServiceNew(host)
	defer dockerClient.Close()
	c, err := dockerClient.Apply(pod, exposedPort)
	a.ds.Register(c)
	return c, err
}

func (a *ApiService) Remove(podName string) error {
	a.ds.UnregisterPod(podName)
	list := a.ds.GetContainersByPod(podName)
	if len(list) == 0 {
		log.Println("No containers found for pod " + podName)
		return errors.New("No containers found for pod " + podName)
	}
	for _, c := range list {
		dockerClient := docker.DockerServiceNew(c.Host)
		defer dockerClient.Close()
		dockerClient.Delete(c.ContainerId)
		a.ds.Unregister(c)
	}
	return nil
}

func (a *ApiService) DeleteContainer(containerName string) error {
	c := a.ds.Discover(containerName)
	dockerClient := docker.DockerServiceNew(c.Host)
	defer dockerClient.Close()
	a.ds.Unregister(c)
	return dockerClient.Delete(c.ContainerId)

}

func (a *ApiService) RestartContainer(containerName string) error {
	c := a.ds.Discover(containerName)
	dockerClient := docker.DockerServiceNew(c.Host)
	defer dockerClient.Close()	
	return dockerClient.Restart(c.ContainerId)

}

func (a *ApiService) RebuildContainer(containerName string) (model.Container, error) {	
	c := a.ds.Discover(containerName)
	pod := a.ds.GetPod(c.Pod)
	dockerClient := docker.DockerServiceNew(c.Host)
	defer dockerClient.Close()
	a.ds.Unregister(c)		
	dockerClient.Delete(c.ContainerId)
	return a.Apply(pod)	
}