package Service

import (
	"encoding/json"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"net/http"
)

func CheckGpuHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	appNum := values.Get("appNum")
	status, error := gpu.CheckGpu(appNum)
	msg := ""
	if error != nil {
		msg = error.Error()
	}
	res := utils.BaseResult{
		Code:   utils.SUCCESS_CODE,
		Status: status,
		Msg:    msg,
	}
	w.Header().Set("Content-Type", "application-json")
	rej, _ := json.Marshal(res)
	w.Write(rej)
}
