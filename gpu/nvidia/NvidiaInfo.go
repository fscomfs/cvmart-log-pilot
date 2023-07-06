package nvidia

import (
	"context"
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
	"log"
	"strconv"
	"strings"
)

type NvidiaInfo struct {
}

func init() {
	defer func() {
		if error := recover(); error != nil {
			log.Printf("nvml init fail %+v", error)
		}
	}()
	r := nvml.Init()
	if r == nvml.SUCCESS {
		gpu.SetExecutor(&NvidiaInfo{})
		log.Printf("nvml init success")
	} else {
		log.Printf("nvml init fail")
	}
}
func (n *NvidiaInfo) ContainerDevices(containerID string) []string {
	var uuids []string
	container, err := gpu.DockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		log.Printf("ContainerInspect containerID=%+v error %+v", containerID, err)
		return uuids
	}
	defer func() {
		if err := recover(); err != nil {
			log.Printf("ContainerDevices error recover:%+v", err)
		}
	}()
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

func (n *NvidiaInfo) Info(indexs []string) (map[string]gpu.InfoObj, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Info error recover:%+v", err)
		}
	}()
	var res = make(map[string]gpu.InfoObj)
	if GpuDeviceMap == nil {
		GpuDeviceMap = make(map[string]nvml.Device)
	}
	for _, v := range indexs {
		var devH nvml.Device
		if _, ok := GpuDeviceMap[v]; ok {
			devH = GpuDeviceMap[v]
		} else {
			if strings.HasPrefix(v, "GPU") {
				devH, re := nvml.DeviceGetHandleByUUID(strings.TrimSpace(v))
				if devH.Handle != nil {
					GpuDeviceMap[v] = devH
				} else {
					nvml.Shutdown()
					nvml.Init()
					return res, fmt.Errorf("get deviceHandle error by uuid:%+v, return:%+v", v, re)
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
		model, _ := nvml.DeviceGetName(devH)
		util, _ := nvml.DeviceGetUtilizationRates(devH)
		res[v] = gpu.InfoObj{
			Total:   memInfo.Total,
			Used:    memInfo.Used,
			GpuType: "GPU",
			Model:   model,
			GpuUtil: util.Gpu,
			MemUtil: util.Memory,
		}
	}
	return res, nil
}
func (n *NvidiaInfo) InfoAll() (map[string]gpu.InfoObj, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Info error recover:%+v", err)
		}
	}()
	count, _ := nvml.DeviceGetCount()
	indexs := []string{}
	if count > 0 {
		for i := 0; i < count; i++ {
			indexs = append(indexs, string(i))
		}
	}
	return n.Info(indexs)
}
