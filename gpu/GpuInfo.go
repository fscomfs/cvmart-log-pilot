package gpu

import (
	"fmt"
	docker "github.com/docker/docker/client"
	"log"
	"os"
)

type GpuInfo interface {
	ContainerDevices(containerID string) []string
	Info(indexs []string) (map[string]InfoObj, error)
	InfoAll() (map[string]InfoObj, error)
}

var executor GpuInfo
var DockerClient *docker.Client

func init() {
	if os.Getenv("DOCKER_API_VERSION") == "" {
		os.Setenv("DOCKER_API_VERSION", "1.23")
	}
	client, err := docker.NewEnvClient()
	if err != nil {
		log.Printf("%+v", err)
	} else {
		DockerClient = client
	}
}

type InfoObj struct {
	Total   uint64 `json:"total"`
	Used    uint64 `json:"used"`
	GpuType string `json:"gpuType"`
	Model   string `json:"model"`
	GpuUtil uint32 `json:"gpuUtil"`
	MemUtil uint32 `json:"memUtil"`
}

func GetExecutor() (GpuInfo, error) {
	if executor == nil {
		return nil, fmt.Errorf("not hava executor")
	}
	return executor, nil
}

func SetExecutor(e GpuInfo) {
	executor = e
}
