package Service

import (
	"encoding/json"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
	_ "github.com/fscomfs/cvmart-log-pilot/gpu/atlas"
	_ "github.com/fscomfs/cvmart-log-pilot/gpu/nvidia"
	_ "github.com/fscomfs/cvmart-log-pilot/gpu/sophgo"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ContainerGpuInfoHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	containerID := values.Get("containerID")
	if containerID == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	gpuInfoExecutor, _ := gpu.GetExecutor()
	var index []string
	if gpuInfoExecutor != nil {
		index = gpuInfoExecutor.ContainerDevices(containerID)
		log.Printf("get device size:%+v,ids:%+v", len(index), index)
	}
	if len(index) > 0 {
		conn, _ := upgrader.Upgrade(w, r, nil)
		defer conn.Close()
		for {
			if res, error := gpuInfoExecutor.Info(index); error != nil {
				log.Printf("get gpu info error:%+v", error)
				return
			} else {
				jsonStr, _ := json.Marshal(res)
				err := conn.WriteMessage(websocket.TextMessage, jsonStr)
				if err != nil {
					log.Printf("-----------gpu info end----------------- err:%+v", err)
					return
				}
			}
			time.Sleep(5 * time.Second)
		}

	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func ContainerGpuInfoForMonitorHandler(w http.ResponseWriter, r *http.Request) {
	if ok, err := RequestAndRedirect(w, r); err != nil || ok {
		return
	}
	values := r.URL.Query()
	containerID := values.Get("containerId")
	if containerID == "" {
		utils.FAIL_RES("containerId is empty", nil, w)
		return
	}
	gpuInfoExecutor, _ := gpu.GetExecutor()
	var index []string
	if gpuInfoExecutor != nil {
		index = gpuInfoExecutor.ContainerDevices(containerID)
		log.Printf("get device size:%+v,ids:%+v", len(index), index)
	}
	if len(index) > 0 {
		if res, error := gpuInfoExecutor.Info(index); error != nil {
			log.Printf("get gpu info error:%+v", error)
			utils.FAIL_RES("get gpu info error", nil, w)
			return
		} else {
			utils.SUCCESS_RES("", res, w)
		}
	} else {
		utils.SUCCESS_RES("no gpu", nil, w)
	}
}
