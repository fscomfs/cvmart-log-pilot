package dsmi

/*
#cgo LDFLAGS: -Wl,--unresolved-symbols=ignore-in-object-files
#include <stdio.h>
#include <stdlib.h>
#include <getopt.h>
#include <unistd.h>
#include "dsmi_common_interface.h"
*/
import "C"
import "fmt"
import "unsafe"

type AtlasInfo struct {
	Total    uint64
	Used     uint64
	CoreRate uint32
	ChipType string
	ChipName string
}

func AllDeviceInfo() (map[int32]AtlasInfo, error) {
	infos := make(map[int32]AtlasInfo)
	deviceCount := 0
	ret := C.dsmi_get_device_count((*C.int)(unsafe.Pointer(&deviceCount)))
	if ret != 0 {
		fmt.Println(deviceCount)
		return infos, fmt.Errorf("device count=0")
	}
	deviceList := make([]int32, deviceCount)
	ret = C.dsmi_list_device((*C.int)(unsafe.Pointer(&deviceList[0])), C.int(deviceCount))
	if ret == 0 {
		for _, v := range deviceList {
			var info AtlasInfo
			var putilization_rate C.uint
			var memInfo C.struct_dsmi_memory_info_stru
			var chipInfo C.struct_dsmi_chip_info_stru
			ret = C.dsmi_get_memory_info(C.int(v), &memInfo)
			if ret == 0 {
				s := C.long(memInfo.utiliza) * C.long(memInfo.memory_size) * C.long(1000*1000) / 100.0
				C.dsmi_get_device_utilization_rate(C.int(v), C.int(2), &putilization_rate)
				C.dsmi_get_chip_info(C.int(v), &chipInfo)
				info = AtlasInfo{
					Total:    uint64(memInfo.memory_size * C.ulonglong(1000*1000)),
					Used:     uint64(s),
					CoreRate: uint32(putilization_rate),
					ChipName: C.GoString((*C.char)(unsafe.Pointer(&chipInfo.chip_name))),
					ChipType: C.GoString((*C.char)(unsafe.Pointer(&chipInfo.chip_type))),
				}
				infos[v] = info
			}
		}
	} else {
		return infos, fmt.Errorf("dsmi_list_device error")
	}
	return infos, nil
}

func GetDeviceInfoByDeviceIds(ids []int32) (map[int32]AtlasInfo, error) {
	infos := make(map[int32]AtlasInfo)
	for _, v := range ids {
		var info AtlasInfo
		var putilization_rate C.uint
		var memInfo C.struct_dsmi_memory_info_stru
		C.dsmi_get_memory_info(C.int(v), &memInfo)
		s := C.long(memInfo.utiliza) * C.long(memInfo.memory_size) / 100.0
		C.dsmi_get_device_utilization_rate(C.int(v), C.int(2), &putilization_rate)
		info = AtlasInfo{
			Total:    uint64(memInfo.memory_size),
			Used:     uint64(s),
			CoreRate: uint32(putilization_rate),
		}
		infos[v] = info
	}
	return infos, nil
}

func GetDeviceCount() (int, error) {
	deviceCount := 0
	ret := C.dsmi_get_device_count((*C.int)(unsafe.Pointer(&deviceCount)))
	if ret == 0 {
		return deviceCount, nil
	} else {
		return deviceCount, fmt.Errorf("dsmi_list_device error")
	}

}
