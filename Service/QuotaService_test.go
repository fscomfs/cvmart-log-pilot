package Service

import (
	"log"
	"os"
	"path"
	"testing"
)

func TestSetDirQuotaHandler(t *testing.T) {
	log.Printf(path.Join("/", "/host/"))

	e := os.Symlink("/1", "/data/tmp")
	if e != nil {
		log.Printf("err %+v", e)
	} else {
		log.Printf("success")
	}

}
