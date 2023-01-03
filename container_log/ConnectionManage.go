package container_log

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"sync"
)

type ConnectHub struct {
	connects map[string]*ConnectDef
}

type ConnectDef struct {
	Id         string     `json:"id"`
	LogClaims  *LogClaims `json:"LogClaims"`
	WriteMsg   chan []byte
	Connect    *websocket.Conn
	closed     bool
	LogMonitor LogMonitor
	ctx        context.Context
	cancel     context.CancelFunc
	mutex      sync.Mutex
}

var connectHub = ConnectHub{
	connects: make(map[string]*ConnectDef),
}

func RegistryConnect(id string, logClaim *LogClaims, conn *websocket.Conn) {
	log.Printf("registry Id:%+v,conn %+v", id, logClaim)
	ctx, cancel := context.WithCancel(context.Background())
	c := ConnectDef{
		Id:        id,
		LogClaims: logClaim,
		Connect:   conn,
		WriteMsg:  make(chan []byte),
		ctx:       ctx,
		cancel:    cancel,
	}
	connectHub.connects[id] = &c

	if logClaim.Operator == OPERATOR_LOG {
		if monitor, err := NewLogMonitor(*logClaim); err == nil {
			c.LogMonitor = monitor
			go monitor.Start(ctx, &c)
		}
	}
	c.watch()
}

func destroy(id string) {
	if _, ok := connectHub.connects[id]; ok {
		connectHub.connects[id].Connect.Close()
		delete(connectHub.connects, id)
	}

}
func (c *ConnectDef) write(message []byte) {
	if c.WriteMsg != nil && !c.closed {
		c.WriteMsg <- message
	}
}
func (c *ConnectDef) watch() {
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				log.Printf("-----------ReadMessage----------------------")
				return
			default:
				messageType, message, err := c.Connect.ReadMessage()
				if err != nil {
					log.Println("Error during message reading:", err)
					c.onClose()
				}
				c.onMessage(messageType, &message)

			}

		}

	}()
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				log.Printf("-----------WriteMessage----------------------")
				return
			case msg := <-c.WriteMsg:
				err := c.Connect.WriteMessage(websocket.BinaryMessage, msg)
				if err != nil {
					log.Printf("WriteMsg err:%+v", err)
					c.onClose()
				}
			}
		}
	}()
}

func (c *ConnectDef) onMessage(messageType int, message *[]byte) {
	if messageType == websocket.CloseMessage {
		c.onClose()
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(*message))
	var msg Message
	if err := decoder.Decode(msg); err != nil {
		log.Printf("message decoder err:%+v", err)
		return
	}
}

func (c *ConnectDef) onClose() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.closed {
		return
	}
	log.Printf("close connect Id:%+v,conn %+v", c.Id, c.LogClaims)
	if c.LogMonitor != nil {
		c.LogMonitor.Close()
	}
	c.closed = true
	c.cancel()
	destroy(c.Id)
}

var LOG_MESSAGE = []byte{'1', '0', '0', '0'}
var RESOURCE_MESSAGE = []byte{'1', '1', '0', '0'}
var GPU_MESSAGE = []byte{'1', '1', '1', '0'}

func logMessage(message []byte) []byte {
	return append(LOG_MESSAGE, message...)
}

func resourceMessage(message []byte) []byte {
	return append(RESOURCE_MESSAGE, message...)
}
func gpuMessage(message []byte) []byte {
	return append(GPU_MESSAGE, message...)
}
