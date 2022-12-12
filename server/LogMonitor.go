package server

type LogMonitor interface {
	Start(def *ConnectDef) error
	Close() error
}

func NewLogMonitor(logClaim LogClaims) (LogMonitor, error) {
	if logClaim.minioObjName == "" {
		if logClaim.Host == "" {
			return NewDockerLog("")
		} else {
			return NewDockerLog(logClaim.Host + ":" + logClaim.Port)
		}

	} else {
		return NewMinioLog("", "")
	}
}
