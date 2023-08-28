package pod_file

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTailFile(t *testing.T) {
	request, _ := http.NewRequest("GET", "http://example.com/some/path", nil)

	_, cancelFunc := context.WithCancel(request.Context())
	go func() {
		//TailFile("/tmp/test.log", nil, *request)
	}()
	time.Sleep(10 * time.Second)

	cancelFunc()

}

func TestReadString(t *testing.T) {
	read := bufio.NewReader(strings.NewReader("nihao\n11111111111111111"))
	for {
		b, e := read.ReadBytes('\n')
		if e != nil {
			log.Printf(string(b))
			break
		}
		log.Printf(string(b))
	}

}
