package main
/*
#cgo CFLAGS: -I./dsmi_common_interface
#cgo LDFLAGS: -L/usr/local/Ascend/driver/lib64/ -l drvdsmi_host
#include <stdio.h>
#include <stdlib.h>
#include <getopt.h>
#include <unistd.h>
#include "dsmi_common_interface.h"
*/
import "C"
import "fmt"
import "unsafe"

type AtlasInfo struct{
	total uint32
	used int32
	coreRate uint32
}
func allDeviceInfo()(map[int32]AtlasInfo) {
	infos :=make(map[int32]AtlasInfo)
	deviceCount:=0
	ret:=C.dsmi_get_device_count((*C.int)(unsafe.Pointer(&deviceCount)))
	if(ret==0){
		fmt.Println(deviceCount)
	}
	deviceList:=make([]int32, deviceCount)
	ret=C.dsmi_list_device((*C.int)(unsafe.Pointer(&deviceList[0])),C.int(8))
	if(ret==0){
		for _,v := range(deviceList) {
			var info AtlasInfo
			var putilization_rate C.uint
			var memInfo C.struct_dsmi_memory_info_stru
			fmt.Println(v)
			C.dsmi_get_memory_info(C.int(v),&memInfo)
			s:=C.long(memInfo.utiliza)*C.long(memInfo.memory_size)/100.0
			C.dsmi_get_device_utilization_rate(C.int(v),C.int(2),&putilization_rate)
			info = AtlasInfo{
				total:uint32(memInfo.memory_size),
				used:int32(s),
				coreRate: uint32(putilization_rate),

			}
			infos[v] = info
		}
	}else{
		fmt.Printf("失败")
	}
	return infos
}

func main(){
	infos:=allDeviceInfo()
	for k,v:=range infos{
		fmt.Printf("K:%+v;%+v\n",k,v)
	}
	
}