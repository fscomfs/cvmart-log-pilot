package server

import (
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/Service"
	"github.com/fscomfs/cvmart-log-pilot/util"
	"net/http"
)

func Handler() {
	http.HandleFunc("/log", Service.LogHandler)
	http.HandleFunc("/api/checkNvidia", Service.CheckGpuHandler)
	http.HandleFunc("/api/containerGpuInfo", Service.ContainerGpuInfoHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", util.ServerPort), nil)
	fmt.Print("ListenAndServe")
}
