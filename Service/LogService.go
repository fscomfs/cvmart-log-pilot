package Service

import (
	"bufio"
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
	url2 "net/url"
	"path/filepath"
	"strings"
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
	values := r.URL.Query()
	token := values.Get("token")
	logParam, err := auth.Auth(token)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	defer func() {
		if err := recover(); err != nil {
			log.Printf("uploadLogByTrackNo error:%+v", err)
		}
	}()
	if logParam.Host == "" {
		//localhost
		if resp, err := utils.GetFileBeatClient().Get(utils.FileBeatUpload + "?trackNo=" + logParam.TrackNo); err == nil {
			content, _ := ioutil.ReadAll(resp.Body)
			re := string(content)
			if re == "1" { //success
				utils.SUCCESS_RES("success", re, w)
				log.Printf("upload success trackNo=%+v", logParam.TrackNo)
			} else { //fail
				utils.SUCCESS_RES("fail", re, w)
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
		if e != nil {
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		url := utils.GetURLByHost(host) + utils.API_UPLOADLOGBYTRACKNO + "?token=" + url2.QueryEscape(t)
		resp, err := utils.GetHttpClient(host).Get(url)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("request uploadLogByTrackNo proxy error %+v", err)
		} else {
			io.Copy(w, resp.Body)
		}
	}

}
