package docker

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/m5m/model"

	"github.com/google/uuid"
)

type DockerService struct {
	host   string
	client *client.Client
}

func DockerServiceNew(dockerHost string) *DockerService {
	host := dockerHost
	client, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		log.Default().Println("Node " + host + " is not available")
	}
	return &DockerService{
		host:   host,
		client: client,
	}
}

func (d *DockerService) Close() {
	// d.client.Close()
}

func (d *DockerService) Inspect(containerId string) (types.ContainerJSON, error) {
	if containerId == "" {
		return types.ContainerJSON{}, errors.New("container ID is empty")
	}
	jsonObject, err := d.client.ContainerInspect(context.Background(), containerId)
	if err != nil {
		log.Println(err)
		log.Println("Error inspecting container " + containerId)
		return types.ContainerJSON{}, err
	}
	jsonObject.Name = strings.Trim(jsonObject.Name, "/")
	return jsonObject, err
}

func (d *DockerService) ListContainers() ([]model.Container, error) {
	containers, err := d.client.ContainerList(context.Background(), container.ListOptions{
		All:    true,
		Latest: true,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var containerList []model.Container
	for _, container := range containers {		
		containerList = append(containerList, model.Container{
			ContainerId:   container.ID,
			Pod:           container.Labels["pod"],
			Port:          container.Labels["port"],
			ContainerPort: container.Labels["containerPort"],
			Image:         container.Image,
			Name:          strings.Trim(strings.Join(container.Names, ""), "/"),
			Status:        container.State,
			Command:       strings.Trim(container.Command, "/"),
			Host:          d.host,
			Age:           time.Since(time.Unix(container.Created, 0)).String(),
			CreatedAt:     container.Created,
		})
	}
	return containerList, nil
}

func (d *DockerService) Delete(containerId string) error {
	if containerId == "" {
		return errors.New("container ID is empty")
	}
	err := d.client.ContainerStop(context.Background(), containerId, container.StopOptions{})
	if err != nil {
		return err
	}
	err = d.client.ContainerRemove(context.Background(), containerId, container.RemoveOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (d *DockerService) Apply(pod model.Pod, exposedPort string) (model.Container, error) {
	if pod.Image == "" {
		return model.Container{}, errors.New("image is empty")
	}
	if pod.Name == "" {
		return model.Container{}, errors.New("name is empty")
	}
	if pod.ContainerPort == "" {
		pod.ContainerPort = "8080"
	}

	ctx := context.Background()

	defaultContainer := model.Container{
		Image:         pod.Image,
		Pod:           pod.Name,
		Name:          pod.Name,
		Host:          d.host,
		ContainerPort: pod.ContainerPort,
		Port:          exposedPort,
		CreatedAt:     time.Now().Unix(),
	}

	reader, err := d.client.ImagePull(ctx, "docker.io/"+pod.Image, image.PullOptions{})
	if err != nil {
		defaultContainer.Status = "ImageCrash"
		return defaultContainer, err
	}

	defer reader.Close()
	io.Copy(os.Stdout, reader)
	name := pod.Name + "-" + uuid.New().String()[:6]
	defaultContainer.Name = name
	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: exposedPort,
	}

	containerPortBinding := nat.PortMap{
		nat.Port(pod.ContainerPort + "/tcp"): []nat.PortBinding{hostBinding},
	}
	resp, err := d.client.ContainerCreate(ctx, &container.Config{
		Image:  pod.Image,
		Tty:    false,
		Labels: map[string]string{"host":d.host,"pod": pod.Name, "port": exposedPort, "containerPort": pod.ContainerPort},
		Cmd:    pod.Cmd,
		ExposedPorts: map[nat.Port]struct{}{
			nat.Port(pod.ContainerPort + "/tcp"): {},
		},
	}, &container.HostConfig{
		PortBindings:    containerPortBinding,
		NetworkMode:     network.NetworkDefault,
		PublishAllPorts: true,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyOnFailure},
	}, nil, nil, name)
	if err != nil {
		defaultContainer.Status = "ContainerError"
		return defaultContainer, err
	}
	if len(resp.Warnings) > 0 {
		defaultContainer.Status = "ContainerWarning"
		log.Println(strings.Join(resp.Warnings, "\n"))
		return defaultContainer, nil
	}
	defaultContainer.ContainerId = resp.ID
	defaultContainer.Status = "ContainerCreating"
	err = d.client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		defaultContainer.Status = "ContainerStartError"
		return defaultContainer, err
	}
	return defaultContainer, nil
}

func (d *DockerService) Restart(containerId string) error {
	if containerId == "" {
		return errors.New("container ID is empty")
	}
	err := d.client.ContainerRestart(context.Background(), containerId, container.StopOptions{})
	if err != nil {
		return err
	}
	return nil
}
