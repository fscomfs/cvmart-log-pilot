package Service

import (
	"encoding/json"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
	_ "github.com/fscomfs/cvmart-log-pilot/gpu/atlas"
	_ "github.com/fscomfs/cvmart-log-pilot/gpu/nvidia"
	_ "github.com/fscomfs/cvmart-log-pilot/gpu/sophgo"
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
		log.Printf("get device executor %+v", len(index))
	}
	if len(index) > 0 {
		conn, _ := upgrader.Upgrade(w, r, nil)
		defer conn.Close()
		for {
			if res, error := gpuInfoExecutor.Info(index); error != nil {
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
