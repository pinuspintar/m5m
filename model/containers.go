package model

import (
	
)

type Container struct {	
	ContainerId   string `json:"containerId"`
	Pod           string `json:"pod"`
	Name          string `json:"name"`
	Image         string `json:"image"`
	Port          string `json:"port"`
	Command       string `json:"command"`
	ContainerPort string `json:"containerPort"`
	Status        string `json:"status"`
	Host          string `json:"host"`
	Age           string `json:"age"`
	CreatedAt     int64 `json:"createdAt"`	
}
