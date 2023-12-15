package Service

import (
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"os"
	"testing"
)

func TestListModelFile(t *testing.T) {
	tarFile := utils.TarFile{Path: "/data/test/test.tar"}
	files := tarFile.ListFiles()
	fmt.Printf("files:%v", files)
	d, _ := os.OpenFile("/tmp/aaa", os.O_CREATE|os.O_RDWR, os.ModePerm)
	tarFile.ExtractFile("./tttt.4", d)

}
