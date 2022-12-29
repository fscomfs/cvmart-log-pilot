package Service

import (
	"encoding/json"
	"github.com/fscomfs/cvmart-log-pilot/gpu"
	"net/http"
)

type CheckRes struct {
	Msg    string `json:"msg"`
	Status int    `json:"status"`
}

func CheckGpuHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	appNum := values.Get("appNum")
	status, error := gpu.CheckGpu(appNum)
	msg := ""
	if error != nil {
		msg = error.Error()
	}
	res := CheckRes{
		Msg:    msg,
		Status: status,
	}
	rej, _ := json.Marshal(res)
	w.Write(rej)
}
