package Service

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/container_log"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"github.com/gorilla/websocket"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/cast"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

type ConnectHub struct {
	connects map[string]*ConnectDef
}

type ConnectDef struct {
	Id         string                   `json:"id"`
	LogClaims  *container_log.LogClaims `json:"LogClaims"`
	WriteMsg   chan []byte
	CloseConn  chan bool
	Connect    *websocket.Conn
	LogMonitor container_log.LogMonitor
}

var connectHub = ConnectHub{
	connects: make(map[string]*ConnectDef),
}

func destroy(id string) {
	connectHub.connects[id].Connect.Close()
	delete(connectHub.connects, id)
}

func LogHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	token := values.Get("token")
	id := values.Get("id")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("RequestHandler upgrader %+v", err)
		return
	}
	//login auth
	logClaims, err := container_log.Auth(token)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("bad auth"))
		conn.WriteMessage(websocket.CloseMessage, []byte(""))
		conn.Close()
		return
	}
	container_log.RegistryConnect(id, logClaims, conn)

}

func DownloadLogHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	obj := values.Get("obj")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("content-Disposition", fmt.Sprintf("attachment;filename=%s", obj+".log"))
	go func() {
		if err := recover(); err != nil {
			log.Printf("download log error %+v", err)
		}
	}()
	if objByte, error := base64.StdEncoding.DecodeString(obj); error == nil {
		object, err := utils.MinioClient.GetObject(r.Context(), utils.Bucket, string(objByte), minio.GetObjectOptions{})
		if err != nil {
			r := bufio.NewReader(object)
			defer object.Close()
			var j interface{}
			for {
				line, e := r.ReadBytes('\n')
				if e != nil {
					if e != io.EOF {
						w.Write([]byte(e.Error() + "\n"))
					}
					return
				}
				e = json.Unmarshal(line, &j)
				if e != nil {
					w.Write([]byte(e.Error() + "\n"))
				}
				data := j.(map[string]interface{})
				if log, ok := data["log"]; ok {
					w.Write([]byte(utils.LineConfound(cast.ToString(log))))
				}
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
			w.Write(content)
			re := string(content)
			if re == "1" { //success
				log.Printf("upload success trackNo=%+v", trackNo)
			} else { //fail
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
