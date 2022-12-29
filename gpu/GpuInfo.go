package gpu

import "fmt"

type GpuInfo interface {
	ContainerDevices(containerID string) []string
	Info(indexs []string) map[string]InfoObj
	InfoAll() map[string]InfoObj
}

var executor GpuInfo

type InfoObj struct {
	Total   uint64 `json:"total"`
	Used    uint64 `json:"used"`
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
