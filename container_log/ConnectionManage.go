package container_log

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
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

func destroy(id string) {
	connectHub.connects[id].Connect.Close()
	delete(connectHub.connects, id)
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
			onMessage(c, messageType, &message)
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
					onClose(c)
					return
				}

			}
		}
	}()
}

func onMessage(c *ConnectDef, messageType int, message *[]byte) {
	if messageType == websocket.CloseMessage {
		destroy(c.Id)
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(*message))
	var msg Message
	if err := decoder.Decode(msg); err != nil {
		log.Printf("message decoder err:%+v", err)
		return
	}
}

func onClose(c *ConnectDef) {
	log.Printf("close connect Id:%+v,conn %+v", c.Id, c.LogClaims)
	if c.LogMonitor != nil {
		c.LogMonitor.Close()
	}
	destroy(c.Id)
}
