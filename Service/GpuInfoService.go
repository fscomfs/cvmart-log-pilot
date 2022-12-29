package Service

import (
	"encoding/json"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
	"github.com/gorilla/websocket"
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
	index := gpuInfoExecutor.ContainerDevices(containerID)
	if len(index) > 0 {
		conn, _ := upgrader.Upgrade(w, r, nil)
		defer conn.Close()
		t := time.NewTicker(1 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				jsonStr, _ := json.Marshal(gpuInfoExecutor.Info(index))
				err := conn.WriteMessage(websocket.TextMessage, jsonStr)
				if err != nil {
					return
				}
			}
		}
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
