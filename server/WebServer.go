package server

import (
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/Service"
	"net/http"
)

func Handler() {
	http.HandleFunc("/log", Service.LogHandler)
	http.HandleFunc("/api/checkNvidia", Service.CheckGpuHandler)
	http.HandleFunc("/api/containerGpuInfo", Service.ContainerGpuInfoHandler)
	http.ListenAndServe(":888", nil)
	fmt.Print("ListenAndServe")
}
