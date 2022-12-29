package gpu

import (
	"context"
	nvml "github.com/NVIDIA/go-nvml/pkg/nvml"
	docker "github.com/docker/docker/client"
	"strconv"
	"strings"
)

type NvidiaInfo struct {
}

var dockerClient *docker.Client

func init() {
	defer func() {
		if error := recover(); error != nil {

		}
	}()
	r := nvml.Init()
	if r == nvml.SUCCESS {
		executor = &NvidiaInfo{}
		dockerClient, _ = docker.NewClient("http://localhost:2375", "", nil, nil)
	}
}
func (n *NvidiaInfo) ContainerDevices(containerID string) []string {
	container, _ := dockerClient.ContainerInspect(context.Background(), containerID)
	var uuids []string
	for _, v := range container.Config.Env {
		if strings.Contains(v, "NVIDIA_VISIBLE_DEVICE") {
			env := strings.Split(v, "=")
			uuids = strings.Split(env[1], ",")
			break
		}
	}
	return uuids
}
func (n *NvidiaInfo) Info(indexs []string) map[string]InfoObj {
	var res = make(map[string]InfoObj)
	for _, v := range indexs {
		var devH nvml.Device
		if strings.HasPrefix(v, "GPU") {
			devH, _ = nvml.DeviceGetHandleByUUID(v)
		} else {
			i, _ := strconv.ParseInt(v, 10, 8)
			devH, _ = nvml.DeviceGetHandleByIndex(int(i))
		}

		memInfo, _ := nvml.DeviceGetMemoryInfo_v2(devH)
		util, _ := nvml.DeviceGetUtilizationRates(devH)
		res[v] = InfoObj{
			Total:   memInfo.Total,
			Used:    memInfo.Used,
			GpuUtil: util.Gpu,
			MemUtil: util.Memory,
		}
	}
	return res
}
func (n *NvidiaInfo) InfoAll() map[string]InfoObj {
	count, _ := nvml.DeviceGetCount()
	indexs := []string{}
	if count > 0 {
		for i := 0; i < count; i++ {
			indexs = append(indexs, string(i))
		}
	}
	return n.Info(indexs)
}
