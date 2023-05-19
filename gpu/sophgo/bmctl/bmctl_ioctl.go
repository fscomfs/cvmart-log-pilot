package bmctl

/*
#cgo LDFLAGS: -Wl,--unresolved-symbols=ignore-in-object-files
#include <stdio.h>
#include <stdlib.h>
#include <getopt.h>
#include <unistd.h>
#include "bm_uapi.h"
*/
import "C"
import (
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/utils/cgoioctl"
	"log"
	"syscall"
	"unsafe"
)

const ioctlMAGIC = 'q'
const DEVICE string = "/host/dev/bmdev-ctl" /* 设备文件*/
func BMCTL_GET_DEV_CNT() cgoioctl.R {
	return cgoioctl.R{Type: ioctlMAGIC, Nr: 0x0, Size: 8}
}

func BMCTL_GET_SMI_ATTR() cgoioctl.WR {
	return cgoioctl.WR{Type: ioctlMAGIC, Nr: 0x01, Size: 8}
}

type DeviceInfo struct {
	DevId    int32
	ChipId   int32
	MemUsed  int64
	MemTotal int64
	TpuUtil  int32
}

func InitCtl() error {
	fd, err := syscall.Open(DEVICE, syscall.O_RDWR, 0777)
	defer func() {
		syscall.Close(fd)
	}()
	if err != nil {
		fmt.Printf("device open failed\r\n")
		syscall.Close(fd)
		fmt.Println(fd, err)
		return err
	}
	return nil
}

func GetAllDeviceInfo() map[int]DeviceInfo {
	infos := make(map[int]DeviceInfo)
	num := 0
	fd, err := syscall.Open(DEVICE, syscall.O_RDWR, 0777)
	defer func() {
		syscall.Close(fd)
	}()
	err = BMCTL_GET_DEV_CNT().Read(fd, unsafe.Pointer(&num))
	if err != nil {
		fmt.Printf("can't set BMCTL_GET_DEV_CNT err=%+v \n", err)
	} else {
		log.Printf("sophgo num %+v", num)
		for i := 0; i < num; i++ {
			var attr C.struct_bm_smi_attr
			attr.dev_id = C.int(i)
			err = BMCTL_GET_SMI_ATTR().Exec(fd, unsafe.Pointer(&attr))
			infos[i] = DeviceInfo{
				DevId:    int32(attr.dev_id),
				ChipId:   int32(attr.chip_id),
				MemUsed:  int64(attr.mem_used) * 1024 * 1024,
				MemTotal: int64(attr.mem_total) * 1024 * 1024,
				TpuUtil:  int32(attr.tpu_util),
			}
		}

	}
	return infos
}
