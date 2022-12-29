package Service

import (
	"github.com/fscomfs/cvmart-log-pilot/container_log"
	"github.com/gorilla/websocket"
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
