package compose

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

type ServiceSpec struct {
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Port       int               `json:"port"`
	Env        map[string]string `json:"env,omitempty"`
	Volumes    []string          `json:"volumes,omitempty"`
	DependsOn  []string          `json:"depends_on,omitempty"`
	HealthPath string            `json:"health_path,omitempty"`
	Domain     string            `json:"domain,omitempty"`
}

type DeploySpec struct {
	Services []ServiceSpec `json:"services"`
}

type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Image       string            `yaml:"image"`
	Ports       []string          `yaml:"ports,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
	Restart     string            `yaml:"restart"`
	Logging     composeLogging    `yaml:"logging"`
}

type composeLogging struct {
	Driver string `yaml:"driver"`
}

func GenerateComposeFile(spec DeploySpec) ([]byte, error) {
	if len(spec.Services) == 0 {
		return nil, errors.New("at least one service is required")
	}
	out := composeFile{Services: map[string]composeService{}}
	for _, s := range spec.Services {
		if s.Name == "" {
			return nil, errors.New("service name is required")
		}
		if s.Image == "" {
			return nil, fmt.Errorf("service %s image is required", s.Name)
		}
		cs := composeService{
			Image:       s.Image,
			Environment: s.Env,
			Volumes:     s.Volumes,
			DependsOn:   s.DependsOn,
			Restart:     "unless-stopped",
			Logging:     composeLogging{Driver: "local"},
		}
		if s.Port > 0 {
			cs.Ports = []string{fmt.Sprintf("%d:%d", s.Port, s.Port)}
		}
		out.Services[s.Name] = cs
	}
	return yaml.Marshal(out)
}
