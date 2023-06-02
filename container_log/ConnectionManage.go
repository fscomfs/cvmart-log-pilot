package container_log

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"github.com/gorilla/websocket"
	"log"
	"sync"
	"time"
)

type ConnectHub struct {
	connects map[string]*ConnectDef
}

type ConnectDef struct {
	Id          string    `json:"id"`
	LogParam    *LogParam `json:"log_param"`
	WriteMsg    chan []byte
	msgCache    map[string][]byte
	timeAfter   *time.Timer
	sendingFlag bool
	Connect     *websocket.Conn
	closed      bool
	LogMonitor  LogMonitor
	ctx         context.Context
	cancel      context.CancelFunc
	mutex       sync.Mutex
}

var connectHub = ConnectHub{
	connects: make(map[string]*ConnectDef),
}

func RegistryConnect(id string, logParam *LogParam, conn *websocket.Conn) {
	log.Printf("registry id:%+v,podLabel:%+v,containerId:%+v,conn %+v", id, logParam.PodLabel, logParam.ContainerId, logParam)
	ctx, cancel := context.WithCancel(context.Background())
	c := ConnectDef{
		Id:          id,
		LogParam:    logParam,
		Connect:     conn,
		sendingFlag: false,
		WriteMsg:    make(chan []byte),
		ctx:         ctx,
		cancel:      cancel,
	}
	connectHub.connects[id] = &c

	if logParam.Operator == OPERATOR_LOG {
		if monitor, err := NewLogMonitor(*logParam); err == nil {
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
func (c *ConnectDef) writeMid(message []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.WriteMsg != nil && !c.closed {
		c.WriteMsg <- message
	}
}
func (c *ConnectDef) write(message []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.msgCache == nil {
		c.msgCache = make(map[string][]byte)
	}
	cacheKey := string(message[0:4])
	_, ok := c.msgCache[cacheKey]
	if ok {
		c.msgCache[cacheKey] = append(c.msgCache[cacheKey], message[4:]...)
	} else {
		c.msgCache[cacheKey] = message
	}
	if !c.sendingFlag {
		c.sendingFlag = true
		if c.timeAfter != nil {
			c.timeAfter.Stop()
		}
		c.timeAfter = time.AfterFunc(600*time.Millisecond, func() {
			c.flush(true)
		})
	}

}

func (c *ConnectDef) flush(lock bool) {
	if lock {
		c.mutex.Lock()
		defer c.mutex.Unlock()
	}
	if c.WriteMsg != nil && !c.closed {
		if len(c.msgCache) > 0 {
			for s, i := range c.msgCache {
				c.WriteMsg <- i
				delete(c.msgCache, s)
			}
		}
	}
	c.sendingFlag = false
}

func (c *ConnectDef) watch() {
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				log.Printf("-----------ReadMessage End----------------------")
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
				log.Printf("-----------WriteMessage End----------------------")
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
	log.Printf("close connect Id:%+v,conn %+v", c.Id, c.LogParam)
	if c.LogMonitor != nil {
		c.LogMonitor.Close()
	}
	c.closed = true
	c.cancel()
	destroy(c.Id)
}

var LOG_MESSAGE = []byte{'1', '0', '0', '0'}
var LOG_STAT_MESSAGE = []byte{'0', '0', '0', '0'}
var RESOURCE_MESSAGE = []byte{'1', '1', '0', '0'}
var GPU_MESSAGE = []byte{'1', '1', '1', '0'}

func logMessage(message []byte) []byte {
	if len(message) > 1 && message[0] == ' ' {
		return append(LOG_MESSAGE, utils.LineConfound(message[1:], false)...)
	}
	return append(LOG_MESSAGE, utils.LineConfound(message, false)...)
}

func logStatMessage(message []byte) []byte {
	return append(LOG_STAT_MESSAGE, message...)
}
func resourceMessage(message []byte) []byte {
	return append(RESOURCE_MESSAGE, message...)
}
func gpuMessage(message []byte) []byte {
	return append(GPU_MESSAGE, message...)
}
