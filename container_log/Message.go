package container_log

type Message struct {
	Type        string `json:"type"`
	Msg         string `json:"msg"`
	ContentType string `json:"content_type"`
}

type ConParam struct {
	operator string `json:"operator"` //CON_TAIL_LOG,CON_HIS_LOG
	follow   bool   `json:"follow"`
	tail     string `json:"tail"` //tail default all 200 300
	start    int    `json:"start"`
	end      int    `json:"end"`
}

const (
	JSON   = "json"
	TXT    = "txt"
	BINARY = "binary"
)

const (
	OPERATOR_LOG = "log"
)
