package container_log

import (
	"context"
	"os"
)

type LogMonitor interface {
	Start(ctx context.Context, def *ConnectDef) error
	Close() error
}

func NewLogMonitor(logClaim LogClaims) (LogMonitor, error) {
	if logClaim.MinioObjName == "" {
		if logClaim.Host == "" {
			return NewDockerLog("")
		} else {
			return NewDockerLog(logClaim.Host + ":" + logClaim.Port)
		}

	} else {
		return NewMinioLog(logClaim.MinioObjName, os.Getenv("BUCKET"))
	}
}