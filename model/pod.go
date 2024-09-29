package model

import "github.com/docker/docker/api/types/strslice"

type Pod struct {
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	ServicePort   string            `json:"servicePort"`
	ContainerPort string            `json:"containerPort"`
	Cmd           strslice.StrSlice `json:"cmd"`
}
