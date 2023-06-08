package mlu

import (
	"context"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
	"github.com/fscomfs/cvmart-log-pilot/gpu/mlu/cndev"
	"github.com/spf13/cast"
	_ "github.com/spf13/cast"
	"log"
	"regexp"
	"strings"
)

var disabledFlag = false

var client cndev.Cndev

type MluInfo struct {
}

func init() {
	defer func() {
		if error := recover(); error != nil {
			log.Printf("mlu init fail %+v", error)
			disabledFlag = true
		}
	}()
	client = cndev.NewCndevClient()
	err := client.Init()
	if err != nil {
		disabledFlag = true
		log.Printf("mlu init fail %+v", err)
	} else {
		gpu.SetExecutor(&MluInfo{})
		log.Printf("mlu init success")
	}
}

func (a *MluInfo) ContainerDevices(containerID string) []string {
	var deviceIds []string
	container, err := gpu.DockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		log.Printf("ContainerInspect containerID=%+v error %+v", containerID, err)
		return deviceIds
	}
	for _, v := range container.HostConfig.Devices {
		if strings.Contains(v.PathOnHost, "/dev/bm-tpu") {
			reg := regexp.MustCompile(`\d+$`)
			res := reg.FindString(v.PathOnHost)
			deviceIds = append(deviceIds, res)
		}
	}
	return deviceIds
}
func (a *MluInfo) Info(indexs []string) (map[string]gpu.InfoObj, error) {
	var res = make(map[string]gpu.InfoObj)
	for _, v := range indexs {
		pyUsed, pyTotal, _, _, err := client.GetDeviceMemory(cast.ToUint(v))
		avgUtil, _, err2 := client.GetDeviceUtil(cast.ToUint(v))
		if err == nil && err2 == nil {
			res[v] = gpu.InfoObj{
				Total:   uint64(pyTotal * 1024 * 1024),
				Used:    uint64(pyUsed * 1024 * 1024),
				GpuUtil: uint32(avgUtil),
				MemUtil: uint32((pyUsed / pyTotal) * 100),
			}
		}
	}

	return res, nil
}
func (a *MluInfo) InfoAll() (map[string]gpu.InfoObj, error) {
	var res = make(map[string]gpu.InfoObj)
	deviceCount, err := client.GetDeviceCount()
	if err == nil && deviceCount > 0 {
		for i := uint(0); i < deviceCount; i++ {
			pyUsed, pyTotal, _, _, err := client.GetDeviceMemory(cast.ToUint(i))
			avgUtil, _, err2 := client.GetDeviceUtil(cast.ToUint(i))
			if err == nil && err2 == nil {
				res[cast.ToString(i)] = gpu.InfoObj{
					Total:   uint64(pyTotal * 1024 * 1024),
					Used:    uint64(pyUsed * 1024 * 1024),
					GpuUtil: uint32(avgUtil),
					MemUtil: uint32((pyUsed / pyTotal) * 100),
				}
			}
		}
	}

	return res, nil
}
