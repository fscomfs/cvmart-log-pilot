package container_log

import (
	"github.com/golang-jwt/jwt/v4"
	"time"
)

type LogParam struct {
	Host           string `json:"host"`
	ContainerId    string `json:"containerId"`
	Operator       string `json:"operator"`
	Tail           string `json:"tail"`
	PodLabel       string `json:"podLabel"`
	MinioObjName   string `json:"minioObjName"`
	ExpirationTime int64  `json:"expirationTime"`
	jwt.RegisteredClaims
}

func (p *LogParam) isContainerLog() bool {
	if p.PodLabel == "" {
		return true
	}
	return false
}

func (p *LogParam) isExpiration() bool {
	if p.ExpirationTime > time.Now().UnixMilli() {
		return false
	}
	return true
}
