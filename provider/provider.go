package provider

import "github.com/m5m/model"

type Provider interface {
	Apply(c model.Container) (model.Container, error)
	Close()
	ListContainers() ([]model.Container, error)
	Remove(c model.Container) error
	HealthCheck(c model.Container) error
	Inspect(containerId string) string
}