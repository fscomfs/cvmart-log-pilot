package Service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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
	"strings"
	"time"
)

type ConnectHub struct {
	connects map[string]*ConnectDef
}

type ConnectDef struct {
	Id         string                  `json:"id"`
	LogParam   *container_log.LogParam `json:"log_param"`
	WriteMsg   chan []byte
	CloseConn  chan bool
	Connect    *websocket.Conn
	LogMonitor container_log.LogMonitor
}

type UploadLogParam struct {
	Token         string `json:"token"`
	Message       string `json:"message"`
	ContainerName string `json:"containerName"`
	MinioObjName  string `json:"minioObjName"`
}

type LocalUploadLogParam struct {
	TrackNo       string `json:"trackNo"`
	Message       string `json:"message"`
	ContainerName string `json:"containerName"`
	MinioObjName  string `json:"minioObjName"`
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
	if !strings.HasPrefix(objName, "/") {
		objName = "/" + objName
	}
	objName = strings.Trim(objName, "\n")
	object, err := utils.GetMinioClient().GetObject(r.Context(), config.GlobConfig.Bucket, objName, minio.GetObjectOptions{})
	if err == nil {
		defer object.Close()
		buffer := make([]byte, 2048)
		r := bufio.NewReader(object)
		for {
			if n, e := r.Read(buffer); e == nil && n > 0 {
				w.Write(buffer[:n])
			} else {
				return
			}
		}
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
		}
		jsonString, err := json.Marshal(p)
		if err != nil {
			log.Printf("uploadLogByTrackNo marshal error %+v", err)
		} //wait 3 second for log cache write
		time.Sleep(3 * time.Second)
		if resp, err := utils.GetFileBeatClient().Post(utils.FileBeatUpload, "application/json", bytes.NewBuffer(jsonString)); err == nil {
			content, _ := ioutil.ReadAll(resp.Body)
			re := string(content)
			if re == "1" { //success
				utils.SUCCESS_RES("success", re, w)
				log.Printf("upload success trackNo=%+v", logParam.TrackNo)
			} else { //fail
				utils.FAIL_RES("fail", re, w)
				log.Printf("upload fail trackNo=%+v", logParam.TrackNo)
			}
		} else {
			log.Printf("request remote uploadFile fail error %+v", err)
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
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		url := utils.GetURLByHost(host) + utils.API_UPLOADLOGBYTRACKNO
		resp, err := utils.GetHttpClient(host).Post(url, "application/json", bytes.NewBuffer(jsonString))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("request uploadLogByTrackNo proxy error %+v", err)
		} else {
			io.Copy(w, resp.Body)
		}
	}

}
