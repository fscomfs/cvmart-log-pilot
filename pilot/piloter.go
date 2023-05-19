package pilot

import (
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"strings"
)

// Global variables for piloter
const (
	PILOT_FILEBEAT = "filebeat"
	PILOT_FLUENTD  = "fluentd"
)

// Piloter interface for piloter
type Piloter interface {
	Name() string

	Start() error
	Reload() error
	Stop() error

	GetBaseConf() string
	GetConfHome() string
	GetConfPath(container string) string

	OnDestroyEvent(container string) error
}

// NewPiloter instantiates a new piloter
func NewPiloter(baseDir string) (Piloter, error) {
	if config.GlobConfig.PilotType == PILOT_FILEBEAT {
		return NewFilebeatPiloter(baseDir)
	}
	if config.GlobConfig.PilotType == PILOT_FLUENTD {
		return NewFluentdPiloter()
	}
	return nil, fmt.Errorf("InvalidPilotType")
}

// CustomConfig custom config
func CustomConfig(name string, customConfigs map[string]string, logConfig *LogConfig) {
	if config.GlobConfig.PilotType == PILOT_FILEBEAT {
		fields := make(map[string]string)
		configs := make(map[string]string)
		for k, v := range customConfigs {
			if strings.HasPrefix(k, name) {
				key := strings.TrimPrefix(k, name+".")
				if strings.HasPrefix(key, "fields") {
					key2 := strings.TrimPrefix(key, "fields.")
					fields[key2] = v
				} else {
					configs[key] = v
				}
			}
		}
		logConfig.CustomFields = fields
		logConfig.CustomConfigs = configs
	}
}
