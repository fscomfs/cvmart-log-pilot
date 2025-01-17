package server

import (
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/Service"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"net/http"
)

func Handler() {
	http.NewServeMux()
	http.HandleFunc(utils.API_LOG, Service.LogHandler)
	http.HandleFunc(utils.API_CHECK_GPU, Service.CheckGpuHandler)
	http.HandleFunc(utils.API_CONTAINERGPUINFO, Service.ContainerGpuInfoHandler)
	http.HandleFunc(utils.API_DOWNLOADLOG, Service.DownloadLogHandler)
	http.HandleFunc(utils.API_UPLOADLOGBYTRACKNO, Service.UploadLogByTrackNo)
	http.HandleFunc(utils.API_SETQUOTA, Service.SetDirQuotaHandler)
	http.HandleFunc(utils.API_GETNODESPACEINFO, Service.GetNodeSpaceInfoHandler)
	http.HandleFunc(utils.API_GETDIRQUOTAINFO, Service.GetDirQuotaInfoHandler)
	http.HandleFunc(utils.API_RELEASEDIR, Service.ReleaseDirHandler)
	http.HandleFunc(utils.API_GETIMAGEQUOTAINFO, Service.GetImageDiskQuotaInfoHandler)
	http.HandleFunc(utils.API_FILES, Service.PodFilesHandler)
	http.HandleFunc(utils.API_FILE, Service.PodFileHandler)
	http.HandleFunc(utils.API_TAIL_FILE, Service.TailFileHandler)
	http.HandleFunc(utils.INTER_TAIL_FILE, Service.TailFileInterHandler)
	http.HandleFunc(utils.API_LIST_MODEL_FILE, Service.ListModelFile)
	http.HandleFunc(utils.API_SAVE_MODEL_FILE, Service.SaveModelFile)
	http.HandleFunc(utils.API_CONTAINTER_GPU_INFO_FOR_MONITOR, Service.ContainerGpuInfoForMonitorHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", config.GlobConfig.ServerPort), nil)
}
