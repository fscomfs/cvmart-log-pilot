package nvidia

import (
	"context"
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
	"github.com/spf13/cast"
	"log"
	"os"
	"os/exec"
	"regexp"
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

	_, e := os.Stat("/host/usr/bin/nvidia-smi")
	if e == nil {
		nvidiasmipath = "/host/usr/bin/nvidia-smi"
	}
	_, e2 := os.Stat("/host/usr/local/bin/nvidia-smi")
	if e2 == nil {
		nvidiasmipath = "/host/usr/local/bin/nvidia-smi"
	}
	_, e3 := os.Stat("/usr/bin/nvidia-smi")
	if e3 == nil {
		nvidiasmipath = "/usr/bin/nvidia-smi"
	}
	_, e4 := os.Stat("/usr/local/bin/nvidia-smi")
	if e4 == nil {
		nvidiasmipath = "/usr/local/bin/nvidia-smi"
	}

	log.Printf("nvidiasmipath:%+v", nvidiasmipath)

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
						uuids = append(uuids, cast.ToString(i))
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
var nvidiasmipath string

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
					gpuInfo, e := Nvidia_smi(v)
					if e == nil {
						res[v] = gpuInfo
						log.Printf("get gpu info by nvidia success")
						continue
					} else {
						log.Printf("get gpu info by nvidia error")
						return res, fmt.Errorf("get deviceHandle error by uuid:%+v, return:%+v", v, re)
					}
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
			indexs = append(indexs, cast.ToString(i))
		}
	}
	return n.Info(indexs)
}

func Nvidia_smi(uuid string) (res gpu.InfoObj, err error) {
	cmd := exec.Command(nvidiasmipath, "--id="+uuid)
	out, e := cmd.CombinedOutput()
	if e != nil {
		log.Printf(e.Error())
		return gpu.InfoObj{}, e
	}
	output := string(out)
	r, _ := GetInfoByString(output)
	return r, nil
}
func GetInfoByString(info string) (res gpu.InfoObj, err error) {
	memorySizeRegex := regexp.MustCompile(`(\d+)MiB`)
	gpuUtilRegex := regexp.MustCompile(`(\d+)%`)
	lines := strings.Split(info, "\n")
	var memorySize uint64
	var used uint64
	var utilRate uint32
	model := "nvidia gpu"
	for i, val := range lines {
		if strings.Contains(val, "===+===") {
			l1 := lines[i+1]
			l1m := strings.Split(l1, "  ")
			model = l1m[2]
			l2 := lines[i+2]
			s := strings.Fields(l2)
			m1 := memorySizeRegex.FindStringSubmatch(s[8])
			if len(m1) >= 2 {
				used = cast.ToUint64(m1[1])
			}
			m2 := memorySizeRegex.FindStringSubmatch(s[10])
			if len(m2) >= 2 {
				memorySize = cast.ToUint64(m2[1])
			}
			m3 := gpuUtilRegex.FindStringSubmatch(s[12])
			if len(m3) >= 2 {
				utilRate = cast.ToUint32(m3[1])
			}
			break
		}
	}
	var memUtil uint32
	if memorySize > 0 {
		memUtil = uint32(((float64(used) * 1024 * 1024) / (float64(memorySize) * 1024 * 1024)) * 100)
	}
	g := gpu.InfoObj{
		Total:   memorySize * 1024 * 1024,
		Used:    used * 1024 * 1024,
		GpuType: "GPU",
		Model:   model,
		MemUtil: memUtil,
		GpuUtil: utilRate,
	}
	return g, nil
}
