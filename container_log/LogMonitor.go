package container_log

import (
	"context"
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"os"
)

type LogMonitor interface {
	Start(ctx context.Context, def *ConnectDef) error
	Close() error
}

func NewLogMonitor(logParam LogParam) (LogMonitor, error) {
	if logParam.MinioObjName == "" {
		if logParam.Host == "" {
			return NewDockerLog("")
		} else {
			return NewDockerLog(logParam.Host + ":" + fmt.Sprint(config.GlobConfig.DockerServerPort))
		}

	} else {
		return NewMinioLog(logParam.MinioObjName, os.Getenv("BUCKET"))
	}
}
