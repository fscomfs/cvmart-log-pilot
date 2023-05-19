package Service

import (
	"bufio"
	"encoding/base64"
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
	obj := values.Get("obj")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("content-Disposition", fmt.Sprintf("attachment;filename=%s", obj+".log"))
	defer func() {
		if err := recover(); err != nil {
			log.Printf("download log error %+v", err)
		}
	}()
	if objByte, error := base64.StdEncoding.DecodeString(obj); error == nil {
		objName := string(objByte)
		if !strings.HasPrefix(objName, "/") {
			objName = "/" + objName
		}
		objName = strings.Trim(objName, "\n")
		_, fileName := filepath.Split(objName)
		if fileName != "" {
			w.Header().Set("content-Disposition", fmt.Sprintf("attachment;filename=%s", fileName))
		} else {
			w.Header().Set("content-Disposition", fmt.Sprintf("attachment;filename=%s", "cvmart-log.log"))
		}
		object, err := utils.MinioClient.GetObject(r.Context(), config.GlobConfig.Bucket, objName, minio.GetObjectOptions{})
		if err == nil {
			r := bufio.NewReader(object)
			defer object.Close()
			for {
				line, e := r.ReadBytes('\n')
				if e != nil {
					if e != io.EOF {
						w.Write([]byte(e.Error() + "\n"))
					}
					return
				}
				w.Write(utils.LineConfound(line, true))
			}
		}
	}
}

func UploadLogByTrackNo(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	trackNo := values.Get("trackNo")
	host := values.Get("host")
	if host == "" {
		//localhost
		if resp, err := utils.FileBeatClient.Get(utils.FileBeatUpload + "?trackNo=" + trackNo); err == nil {
			content, _ := ioutil.ReadAll(resp.Body)
			re := string(content)
			if re == "1" { //success
				utils.SUCCESS_RES("success", re, w)
				log.Printf("upload success trackNo=%+v", trackNo)
			} else { //fail
				utils.SUCCESS_RES("fail", re, w)
				log.Printf("upload fail trackNo=%+v", trackNo)
			}
		} else {
			log.Printf("request remote uploadFile fail error %+v", err)
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		resp, err := utils.ProxyHttpClient.Get(utils.GetURLByHost(host) + utils.API_UPLOADLOGBYTRACKNO + "?trackNo=" + trackNo)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("request uploadLogByTrackNo proxy error %+v", err)
		} else {
			io.Copy(w, resp.Body)
		}
	}

}
