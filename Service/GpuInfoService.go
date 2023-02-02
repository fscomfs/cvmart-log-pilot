package Service

import (
	"encoding/json"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
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
	gpuInfoExecutor, _ := gpu.GetExecutor()
	var index []string
	if gpuInfoExecutor != nil {
		index = gpuInfoExecutor.ContainerDevices(containerID)
	}
	if len(index) > 0 {
		log.Printf("device:%+v", index)
		conn, _ := upgrader.Upgrade(w, r, nil)
		defer conn.Close()
		for {
			if res, error := gpuInfoExecutor.Info(index); error != nil {
				return
			} else {
				jsonStr, _ := json.Marshal(res)
				err := conn.WriteMessage(websocket.TextMessage, jsonStr)
				if err != nil {
					return
				}
			}
			time.Sleep(2 * time.Second)
		}

	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
