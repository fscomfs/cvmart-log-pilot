package gpu

import (
	"context"
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	docker "github.com/docker/docker/client"
	"log"
	"os"
	"strconv"
	"strings"
)

type NvidiaInfo struct {
}

var dockerClient *docker.Client

func init() {
	defer func() {
		if error := recover(); error != nil {
			log.Printf("nvml init fail %+v", error)
		}
	}()
	r := nvml.Init()
	if r == nvml.SUCCESS {
		executor = &NvidiaInfo{}
		IP := os.Getenv("NODE_IP")
		if IP == "" {
			IP = "localhost"
		}
		dockerClient, _ = docker.NewClient("http://"+IP+":2375", "", nil, nil)
	} else {
		log.Printf("nvml init fail")
	}
}
func (n *NvidiaInfo) ContainerDevices(containerID string) []string {
	var uuids []string
	container, err := dockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		log.Printf("ContainerInspect containerID=%+v error %+v", containerID, err)
		return uuids
	}
	for _, v := range container.Config.Env {
		if strings.Contains(v, "NVIDIA_VISIBLE_DEVICE") {
			env := strings.Split(v, "=")
			if env[1] == "all" {
				count, _ := nvml.DeviceGetCount()
				if count > 0 {
					for i := 0; i < count; i++ {
						uuids = append(uuids, string(i))
					}
				}
				break
			}
			uuids = strings.Split(env[1], ",")
			break
		}
	}
	return uuids
}

var GpuDeviceMap map[string]nvml.Device

func (n *NvidiaInfo) Info(indexs []string) (map[string]InfoObj, error) {
	var res = make(map[string]InfoObj)
	if GpuDeviceMap == nil {
		GpuDeviceMap = make(map[string]nvml.Device)
	}
	for _, v := range indexs {
		var devH nvml.Device
		if _, ok := GpuDeviceMap[v]; ok {
			devH = GpuDeviceMap[v]
		} else {
			if strings.HasPrefix(v, "GPU") {
				devH, _ = nvml.DeviceGetHandleByUUID(v)
				if devH.Handle != nil {
					GpuDeviceMap[v] = devH
				} else {
					return res, fmt.Errorf("get deviceHandle error")
				}
			} else {
				i, _ := strconv.ParseInt(v, 10, 8)
				devH, _ = nvml.DeviceGetHandleByIndex(int(i))
				if devH.Handle != nil {
					GpuDeviceMap[v] = devH
				} else {
					return res, fmt.Errorf("get deviceHandle error")
				}
			}
		}
		memInfo, _ := nvml.DeviceGetMemoryInfo(devH)
		util, _ := nvml.DeviceGetUtilizationRates(devH)
		res[v] = InfoObj{
			Total:   memInfo.Total,
			Used:    memInfo.Used,
			GpuUtil: util.Gpu,
			MemUtil: util.Memory,
		}
	}
	return res, nil
}
func (n *NvidiaInfo) InfoAll() (map[string]InfoObj, error) {
	count, _ := nvml.DeviceGetCount()
	indexs := []string{}
	if count > 0 {
		for i := 0; i < count; i++ {
			indexs = append(indexs, string(i))
		}
	}
	return n.Info(indexs)
}