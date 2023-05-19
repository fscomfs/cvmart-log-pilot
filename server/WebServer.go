package server

import (
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/Service"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"net/http"
)

func Handler() {
	http.HandleFunc(utils.API_LOG, Service.LogHandler)
	http.HandleFunc(utils.API_CHECK_GPU, Service.CheckGpuHandler)
	http.HandleFunc(utils.API_CONTAINERGPUINFO, Service.ContainerGpuInfoHandler)
	http.HandleFunc(utils.API_DOWNLOADLOG, Service.DownloadLogHandler)
	http.HandleFunc(utils.API_UPLOADLOGBYTRACKNO, Service.UploadLogByTrackNo)
	http.ListenAndServe(fmt.Sprintf(":%d", config.GlobConfig.ServerPort), nil)
}
