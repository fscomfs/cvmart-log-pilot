package gpu

import (
	"context"
	docker "github.com/docker/docker/client"
	"github.com/fscomfs/cvmart-log-pilot/gpu/dcmi"
	"github.com/spf13/cast"
	_ "github.com/spf13/cast"
	"log"
	"os"
	"strings"
)

type AtlasInfo struct {
}

func init() {
	defer func() {
		if error := recover(); error != nil {
			log.Printf("dcmi init fail %+v", error)
		}
	}()

	dcmi.Init()
	if count, error := dcmi.GetDeviceCount(); error != nil {
		log.Printf("dcmi init error %+v", error)
	} else {
		if count > 0 {
			IP := os.Getenv("NODE_IP")
			executor = &AtlasInfo{}
			if IP == "" {
				IP = "localhost"
			}
			dockerClient, _ = docker.NewClient("http://"+IP+":2375", "", nil, nil)
		}
	}
}

func (a *AtlasInfo) ContainerDevices(containerID string) []string {
	var deviceIds []string
	container, err := dockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		log.Printf("ContainerInspect containerID=%+v error %+v", containerID, err)
		return deviceIds
	}
	for _, v := range container.Config.Env {
		if strings.Contains(v, "ASCEND_VISIBLE_DEVICES") {
			env := strings.Split(v, "=")
			deviceIds = strings.Split(env[1], ",")
			break
		}
	}
	return deviceIds
}
func (a *AtlasInfo) Info(indexs []string) (map[string]InfoObj, error) {
	var res = make(map[string]InfoObj)
	ids := []int32{}
	for i := range indexs {
		ids = append(ids, cast.ToInt32(indexs[i]))
	}
	if allInfo, error := dcmi.GetDeviceInfoByDeviceIds(ids); error != nil {
		for k := range allInfo {
			res[string(k)] = InfoObj{
				Total:   uint64(allInfo[k].Total),
				Used:    uint64(allInfo[k].Used),
				GpuUtil: allInfo[k].CoreRate,
				MemUtil: uint32(allInfo[k].Used) / allInfo[k].Total,
			}
		}
	} else {
		return res, error
	}
	return res, nil
}
func (a *AtlasInfo) InfoAll() (map[string]InfoObj, error) {
	var res = make(map[string]InfoObj)
	if allInfo, error := dcmi.AllDeviceInfo(); error != nil {
		for k := range allInfo {
			res[string(k)] = InfoObj{
				Total:   uint64(allInfo[k].Total),
				Used:    uint64(allInfo[k].Used),
				GpuUtil: allInfo[k].CoreRate,
				MemUtil: uint32(allInfo[k].Used) / allInfo[k].Total,
			}
		}
	} else {
		return res, error
	}
	return res, nil
}
