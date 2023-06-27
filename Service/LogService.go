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
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"net/http"
	"path/filepath"
	"strings"
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
	PodLabel      string `json:"podLabel"`
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

	if logParam.Host == "localhost" {
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
			retry.Delay(20*time.Second),
		)
		if requestError != nil {
			log.Printf("request remote uploadFile fail error %+v", err)
			utils.FAIL_RES(requestError.Error(), nil, w)
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		host := logParam.Host
		logParam.Host = "localhost"
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
		resData := &UploadByParamRes{
			TrackFlag:  0,
			UploadCode: 0,
		}
		doProxySaveFunc := func(pod *coreV1.Pod) {
			url := utils.GetURLByHost(host) + utils.API_UPLOADLOGBYTRACKNO
			if resData.TrackFlag == 1 {
				time.Sleep(5 * time.Second)
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
					retry.Delay(20*time.Second),
				)
				if requestError != nil {
					var resBody []byte
					if param.Async == 0 {
						resBody = utils.FAIL_RES(requestError.Error(), nil, w)
						w.WriteHeader(http.StatusBadRequest)
					} else {
						resBody = utils.FAIL_RES(requestError.Error(), nil, nil)
					}
					go saveLogCallback(param.CallBackUrl, 0, resBody)
					log.Printf("request uploadLogByTrackNo proxy error %+v", err)
				}
			} else {
				if pod != nil {
					tailLimes := int64(100000)
					limitBytes := int64(100 * 1024 * 1024)
					req := utils.GetK8sClient().CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &coreV1.PodLogOptions{
						TailLines:  &tailLimes,
						LimitBytes: &limitBytes,
					})
					errInfoMessage := ""
					appendMessageReader := bytes.NewReader([]byte("\n" + param.Message))
					if logReader, errLog := req.Stream(context.Background()); errLog == nil {
						uploadInfo, uploadErr := utils.GetMinioClient().PutObject(context.Background(), config.GlobConfig.Bucket, param.MinioObjName, io.MultiReader(logReader, appendMessageReader), -1, minio.PutObjectOptions{
							ContentType: "application/octet-stream",
						})
						if uploadErr != nil {
							resData.UploadCode = 0
							errInfoMessage = uploadErr.Error()
							log.Printf("no track upload k8s api log error:%+v", uploadErr)
						} else {
							resData.UploadCode = 1
							log.Printf("no track upload k8s api log success:%+v", uploadInfo)
						}
					} else {
						if _, errInfo := container_log.GetPodErrorInfo(pod.Status); errInfo != nil {
							uploadInfo, uploadErr := utils.GetMinioClient().PutObject(context.Background(), config.GlobConfig.Bucket, param.MinioObjName, io.MultiReader(bytes.NewReader([]byte(errInfo.Error())), appendMessageReader), -1, minio.PutObjectOptions{
								ContentType: "application/octet-stream",
							})
							if uploadErr != nil {
								errInfoMessage = uploadErr.Error()
								resData.UploadCode = 0
								log.Printf("no track upload k8s api log error:%+v", uploadErr)
							} else {
								resData.UploadCode = 1
								log.Printf("no track upload k8s api log success:%+v", uploadInfo)
							}
						}
					}
					var resBody []byte
					if resData.UploadCode == 1 {
						resBody = utils.SUCCESS_RES("success", resData, nil)
					} else {
						resBody = utils.FAIL_RES(errInfoMessage, resData, nil)
					}
					go saveLogCallback(param.CallBackUrl, 1, resBody)
				}
			}

		}
		//async exec
		listOption := v1.ListOptions{
			Watch:         false,
			LabelSelector: "app=" + param.PodLabel,
		}
		podList, err := utils.GetK8sClient().CoreV1().Pods("default").List(context.Background(), listOption)
		isTrack := 0
		var pod *coreV1.Pod = nil
		if err == nil && len(podList.Items) > 0 {
			pod = &podList.Items[0]
			if host == "" {
				host = pod.Status.HostIP
			}
		loop1:
			for _, container := range pod.Spec.Containers {
				for _, envVar := range container.Env {
					if strings.Contains(envVar.Name, "cvmart_logs_stdout") {
						isTrack = 1
						break loop1
					}
				}
			}
		} else {
			isTrack = 1
		}
		if host == "" {
			host = "localhost"
		}
		resData.TrackFlag = isTrack
		if param.Async > 0 {
			go doProxySaveFunc(pod)
			utils.SUCCESS_RES("request success", resData, w)
		} else {
			doProxySaveFunc(pod)
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
