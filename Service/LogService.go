package Service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"github.com/fscomfs/cvmart-log-pilot/container_log"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/minio/minio-go/v7"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

type ConnectHub struct {
	connects map[string]*ConnectDef
}

type ConnectDef struct {
	Id         string                  `json:"id"`
	LogParam   *container_log.LogParam `json:"logParam"`
	WriteMsg   chan []byte
	CloseConn  chan bool
	Connect    *websocket.Conn
	LogMonitor container_log.LogMonitor
}
type UploadByParamRes struct {
	TrackFlag  int `json:"trackFlag"`
	UploadCode int `json:"uploadCode"`
}

type UploadLogParam struct {
	Token         string `json:"token"`
	Message       string `json:"message"`
	ContainerName string `json:"containerName"`
	MinioObjName  string `json:"minioObjName"`
	CallBackUrl   string `json:"callBackUrl"`
	Async         int    `json:"async"`
}

type LocalUploadLogParam struct {
	TrackNo       string `json:"trackNo"`
	Message       string `json:"message"`
	ContainerName string `json:"containerName"`
	MinioObjName  string `json:"minioObjName"`
	CallBackUrl   string `json:"callBackUrl"`
	Async         int    `json:"async"`
}

var connectHub = ConnectHub{
	connects: make(map[string]*ConnectDef),
}

var auth = container_log.AESAuth{}

func destroy(id string) {
	connectHub.connects[id].Connect.Close()
	delete(connectHub.connects, id)
}

func LogHandler(w http.ResponseWriter, r *http.Request) {
	uuid, _ := uuid.NewUUID()
	id := uuid.String()
	token := r.URL.Query().Get("token")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("RequestHandler upgrader %+v", err)
		return
	}
	//login auth
	logParam, err := auth.Auth(token)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		conn.WriteMessage(websocket.CloseMessage, []byte("connect close"))
		conn.Close()
		return
	}
	container_log.RegistryConnect(id, logParam, conn)

}

func DownloadLogHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	token := values.Get("token")
	logParam, err := auth.Auth(token)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusUnauthorized)
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("content-Disposition", fmt.Sprintf("attachment;filename=%s", filepath.Base(logParam.MinioObjName)))
	defer func() {
		if err := recover(); err != nil {
			log.Printf("download log error %+v", err)
		}
	}()
	objName := logParam.MinioObjName
	resObj, err2 := utils.GetMinioClient().GetObject(context.Background(), config.GlobConfig.Bucket, objName, minio.GetObjectOptions{})
	if err2 == nil {
		defer resObj.Close()
		io.Copy(w, resObj)
	} else {
		w.Write([]byte(err.Error()))
	}
}

func UploadLogByTrackNo(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("uploadLogByTrackNo error:%+v", err)
		}
	}()
	param := UploadLogParam{}
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		log.Printf("upload log parse param fail %+v", err)
	}

	logParam, err := auth.Auth(param.Token)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if logParam.Host == "" {
		//localhost
		p := LocalUploadLogParam{
			TrackNo:       logParam.TrackNo,
			Message:       param.Message,
			ContainerName: param.ContainerName,
			MinioObjName:  param.MinioObjName,
			CallBackUrl:   param.CallBackUrl,
			Async:         param.Async,
		}
		jsonString, err := json.Marshal(p)
		if err != nil {
			log.Printf("uploadLogByTrackNo marshal error %+v", err)
		}
		requestError := retry.Do(func() error {
			resp, err2 := utils.GetFileBeatClient().Post(utils.FileBeatUpload, "application/json", bytes.NewBuffer(jsonString))
			if err2 == nil {
				var res UploadByParamRes
				if err2 = json.NewDecoder(resp.Body).Decode(&res); err2 == nil {
					utils.SUCCESS_RES("success", res, w)
				}
			}
			return err2
		},
			retry.Attempts(3),
			retry.Delay(10*time.Second),
		)
		if requestError != nil {
			log.Printf("request remote uploadFile fail error %+v", err)
			utils.FAIL_RES(err.Error(), nil, w)
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		host := logParam.Host
		logParam.Host = ""
		t, e := auth.GeneratorToken(*logParam)
		param.Token = t
		jsonString, err := json.Marshal(param)
		if err != nil {
			log.Printf("uploadLogByTrackNo marshal error %+v", err)
		}
		if e != nil {
			utils.FAIL_RES(err.Error(), nil, w)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		url := utils.GetURLByHost(host) + utils.API_UPLOADLOGBYTRACKNO
		doProxySaveFunc := func() {
			time.Sleep(3 * time.Second)
			requestError := retry.Do(func() error {
				resp, err2 := utils.GetHttpClient(host).Post(url, "application/json", bytes.NewBuffer(jsonString))
				if err2 == nil {
					r, _ := ioutil.ReadAll(resp.Body)
					if param.Async == 0 {
						io.Copy(w, bytes.NewReader(r))
					}
					go saveLogCallback(param.CallBackUrl, 1, r)
				}
				return err2
			},
				retry.Attempts(3),
				retry.Delay(10*time.Second),
			)
			if requestError != nil {
				var resBody []byte
				if param.Async == 0 {
					resBody = utils.FAIL_RES(err.Error(), nil, w)
					w.WriteHeader(http.StatusBadRequest)
				}
				go saveLogCallback(param.CallBackUrl, 0, resBody)
				log.Printf("request uploadLogByTrackNo proxy error %+v", err)
			}
		}
		//async exec
		if param.Async > 0 {
			go doProxySaveFunc()
			utils.SUCCESS_RES("request success", nil, w)
		} else {
			doProxySaveFunc()
		}

	}
}

func saveLogCallback(url string, status int, res []byte) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("callback request error %+v", err)
		}
	}()
	if url != "" {
		log.Printf("do callback url:%+v,--status:%+v", url, status)
		utils.GetRetryHttpClient().Post(url, "application/json", res)
	}
}
