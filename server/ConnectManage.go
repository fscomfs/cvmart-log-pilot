package server

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type ConnectHub struct {
	connects map[string]*ConnectDef
}

type ConnectDef struct {
	Id         string     `json:"id"`
	LogClaims  *LogClaims `json:"LogClaims"`
	WriteMsg   chan []byte
	CloseConn  chan bool
	Connect    *websocket.Conn
	LogMonitor LogMonitor
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var connectHub = ConnectHub{
	connects: make(map[string]*ConnectDef),
}

func RegistryConnect(id string, logClaim *LogClaims, conn *websocket.Conn) {
	log.Printf("registry Id:%+v,conn %+v", id, logClaim)
	c := ConnectDef{
		Id:        id,
		LogClaims: logClaim,
		Connect:   conn,
		WriteMsg:  make(chan []byte),
		CloseConn: make(chan bool),
	}
	connectHub.connects[id] = &c
	if logClaim.Operator == OPERATOR_LOG {
		if monitor, err := NewLogMonitor(*logClaim); err == nil {
			c.LogMonitor = monitor
			go monitor.Start(&c)
		}
	}
	c.watch()
}

func Destroy(id string) {
	connectHub.connects[id].Connect.Close()
	delete(connectHub.connects, id)
}

func RequestHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	token := values.Get("token")
	id := values.Get("id")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("RequestHandler upgrader %+v", err)
		return
	}
	//login auth
	logClaims, err := Auth(token)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("bad auth"))
		conn.WriteMessage(websocket.CloseMessage, []byte(""))
		conn.Close()
		return
	}
	RegistryConnect(id, logClaims, conn)

}

func (c *ConnectDef) watch() {
	go func() {
		for {
			messageType, message, err := c.Connect.ReadMessage()
			if err != nil {
				log.Println("Error during message reading:", err)
				c.CloseConn <- true
				return
			}
			OnMessage(c, messageType, &message)
		}

	}()
	go func() {
		for {
			select {
			case msg := <-c.WriteMsg:
				err := c.Connect.WriteMessage(websocket.BinaryMessage, msg)
				if err != nil {
					log.Fatalf("WriteMsg err:%+v", err)
					c.CloseConn <- true
				}
			case close := <-c.CloseConn:
				if close {
					OnClose(c)
					return
				}

			}
		}
	}()
}

func OnMessage(c *ConnectDef, messageType int, message *[]byte) {
	if messageType == websocket.CloseMessage {
		Destroy(c.Id)
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(*message))
	var msg Message
	if err := decoder.Decode(msg); err != nil {
		log.Printf("message decoder err:%+v", err)
		return
	}
	//if msg.Type == MSG_CONTROLLER {
	//	var msgBody ConParam
	//	msgBodyDecoder := json.NewDecoder(bytes.NewReader([]byte(msg.Msg)))
	//	if err := msgBodyDecoder.Decode(msgBody); err != nil {
	//		log.Printf("message decoder err:%+v", err)
	//		return
	//	}
	//	if msgBody.operator == CON_TAIL_LOG {
	//
	//	}
	//
	//}

}

func OnClose(c *ConnectDef) {
	log.Printf("close connect Id:%+v,conn %+v", c.Id, c.LogClaims)
	if c.LogMonitor != nil {
		c.LogMonitor.Close()
	}
	Destroy(c.Id)
}
