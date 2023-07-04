package sophgo

import (
	"context"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
	"github.com/fscomfs/cvmart-log-pilot/gpu/sophgo/bmctl"
	"github.com/spf13/cast"
	_ "github.com/spf13/cast"
	"log"
	"regexp"
	"strings"
)

type SophgoInfo struct {
}

var disabledFlag = false

func init() {
	defer func() {
		if error := recover(); error != nil {
			log.Printf("sophgo init fail %+v", error)
			disabledFlag = true
		}
	}()
	if err := bmctl.InitCtl(); err == nil {
		gpu.SetExecutor(&SophgoInfo{})
		log.Printf("sophgo init success")
	} else {
		log.Printf("sophgo init fail %+v", err)
	}

}

func (a *SophgoInfo) ContainerDevices(containerID string) []string {
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
func (a *SophgoInfo) Info(indexs []string) (map[string]gpu.InfoObj, error) {
	var res = make(map[string]gpu.InfoObj)
	allInfo := bmctl.GetAllDeviceInfo()
	for _, v := range allInfo {
		e := false
		for _, index := range indexs {
			if index == cast.ToString(v.DevId) {
				e = true
			}
		}
		if e {
			//var MemUtil uint32
			//if v.MemTotal > 0 {
			//	MemUtil = uint32((v.MemUsed / v.MemTotal) * 100)
			//}
			res[cast.ToString(v.DevId)] = gpu.InfoObj{
				//Total:   uint64(v.MemTotal),
				//Used:    uint64(v.MemUsed),
				GpuUtil: uint32(v.TpuUtil),
				GpuType: "TPU",
				Model:   "BM1684",
				//MemUtil: MemUtil,
			}
		}

	}

	return res, nil
}
func (a *SophgoInfo) InfoAll() (map[string]gpu.InfoObj, error) {
	var res = make(map[string]gpu.InfoObj)
	allInfo := bmctl.GetAllDeviceInfo()
	for _, v := range allInfo {
		res[cast.ToString(v.DevId)] = gpu.InfoObj{
			Total:   uint64(v.MemTotal),
			Used:    uint64(v.MemUsed),
			GpuUtil: uint32(v.TpuUtil),
			MemUtil: uint32((v.MemUsed / v.MemTotal) * 100),
		}
	}
	return res, nil
}
