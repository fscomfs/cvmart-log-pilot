package atlas

import (
	"context"
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
	"github.com/fscomfs/cvmart-log-pilot/gpu/atlas/common"
	"github.com/fscomfs/cvmart-log-pilot/gpu/atlas/dcmi"
	"github.com/fscomfs/cvmart-log-pilot/gpu/atlas/dsmi"
	"github.com/spf13/cast"
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
func (a *AtlasInfo) Info(indexs []string) (res map[string]gpu.InfoObj, reserror error) {
	res = make(map[string]gpu.InfoObj)
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Info error recover %+v", err)
			reserror = fmt.Errorf("Info error recover")
		}
	}()
	all, err := a.InfoAll()
	//log.Printf("all info indexs:%+v,all:%+v", indexs, all)
	if err != nil {
		log.Printf("all info error: %+v", err)
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
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Info All error recover %+v", err)
		}
	}()
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
					memoryInfo, errinfo := dc.DcGetMemoryInfo(carIndex, deviceId)
					if errinfo != nil {
						log.Printf("dcmi get memory info error carIndex=%+v,deviceId=%+v,error info=%+v", carIndex, deviceId, errinfo)
						continue
					}
					coreRate, _ := dc.DcGetDeviceUtilizationRate(carIndex, deviceId, common.AICore)
					info, _ := dc.DcGetChipInfo(carIndex, deviceId)
					res[cast.ToString(index)] = gpu.InfoObj{
						Total:   memoryInfo.MemorySize * uint64(1000*1000),
						Used:    (memoryInfo.MemorySize - memoryInfo.MemoryAvailable) * uint64(1000*1000),
						GpuType: "NPU",
						Model:   info.Name,
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
				res[cast.ToString(k)] = gpu.InfoObj{
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
