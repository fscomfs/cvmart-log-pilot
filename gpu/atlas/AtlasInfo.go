package atlas

import (
	"context"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
	"github.com/fscomfs/cvmart-log-pilot/gpu/atlas/common"
	"github.com/fscomfs/cvmart-log-pilot/gpu/atlas/dcmi"
	"github.com/fscomfs/cvmart-log-pilot/gpu/atlas/dsmi"
	_ "github.com/spf13/cast"
	"log"
	"strings"
)

var dc *dcmi.DcManager

type AtlasInfo struct {
	dcmi bool
}

var disabledFlag = false

func init() {
	defer func() {
		if error := recover(); error != nil {
			log.Printf("dsmi init fail %+v", error)
			disabledFlag = true
		}
	}()
	dc = &dcmi.DcManager{}
	defer dc.DcShutDown()
	err := dc.DcInit()
	if err == nil {
		gpu.SetExecutor(&AtlasInfo{true})
		log.Printf("dcmi init success")
	} else {
		dsmi.Init()
		gpu.SetExecutor(&AtlasInfo{false})
		log.Printf("dsmi init success")
	}
}

func (a *AtlasInfo) ContainerDevices(containerID string) []string {
	var deviceIds []string
	container, err := gpu.DockerClient.ContainerInspect(context.Background(), containerID)
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
func (a *AtlasInfo) Info(indexs []string) (map[string]gpu.InfoObj, error) {
	var res = make(map[string]gpu.InfoObj)
	all, err := a.InfoAll()
	if err != nil {
		return res, err
	}
	for s, obj := range all {
		for _, index := range indexs {
			if index == s {
				res[index] = obj
			}
		}
	}
	return res, nil
}
func (a *AtlasInfo) InfoAll() (map[string]gpu.InfoObj, error) {
	var res = make(map[string]gpu.InfoObj)
	if a.dcmi {
		dc.DcInit()
		defer dc.DcShutDown()
		deviceCount, r := dc.DcGetDeviceCount()
		if r == nil && deviceCount > 0 {
			_, carList, _ := dc.DcGetCardList()
			index := 0
			for _, carIndex := range carList {
				deviceIdMax, _ := dc.DcGetDeviceNumInCard(carIndex)
				for deviceId := int32(0); deviceId < deviceIdMax; deviceId++ {
					memoryInfo, _ := dc.DcGetMemoryInfo(carIndex, deviceId)
					coreRate, _ := dc.DcGetDeviceUtilizationRate(carIndex, deviceId, common.AICore)
					res[string(index)] = gpu.InfoObj{
						Total:   memoryInfo.MemorySize * uint64(1000*1000),
						Used:    (memoryInfo.MemorySize - memoryInfo.MemoryAvailable) * uint64(1000*1000),
						GpuUtil: uint32(coreRate),
						MemUtil: memoryInfo.Utilization,
					}
					index++
				}
			}
		} else {
			return res, r
		}

	} else {
		dsmi.Init()
		defer dsmi.Shutdown()
		if allInfo, error := dsmi.AllDeviceInfo(); error != nil {
			for k := range allInfo {
				res[string(k)] = gpu.InfoObj{
					Total:   allInfo[k].Total,
					Used:    allInfo[k].Used,
					GpuUtil: allInfo[k].CoreRate,
					MemUtil: uint32(allInfo[k].Used / allInfo[k].Total),
				}
			}
		} else {
			return res, error
		}
	}

	return res, nil
}