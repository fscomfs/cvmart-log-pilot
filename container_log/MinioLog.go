package container_log

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"os"
)

var minioClient *minio.Client

type MinioLog struct {
	minioObjectName string `json:"minio_object_name"`
	bucketName      string `json:"bucket_name"`
	closed          bool   `json:"closed"`
}

func (m *MinioLog) Start(ctx context.Context, def *ConnectDef) error {
	if minioClient == nil {
		minioClient, _ = newMinioClient()

	}
	object, err := minioClient.GetObject(ctx, m.bucketName, m.minioObjectName, minio.GetObjectOptions{})
	if err != nil {
		def.WriteMsg <- []byte(err.Error() + "\n")
		m.closed = true
		return err
	}

	r := bufio.NewReader(object)
	defer object.Close()
	var j interface{}
	for {
		line, e := r.ReadBytes('\n')
		if e != nil {
			if e != io.EOF {
				def.WriteMsg <- []byte(e.Error() + "\n")
			}
			return e
		}
		e = json.Unmarshal(line, &j)
		if e != nil {
			def.WriteMsg <- []byte(e.Error() + "\n")
			return e
		}
		data := j.(map[string]interface{})
		if log, ok := data["log"]; ok {
			def.WriteMsg <- []byte(log.(string))
		}
		if m.closed {
			return nil
		}
	}
	return nil
}
func (m *MinioLog) Close() error {
	m.closed = true
	return nil
}

func NewMinioLog(objectName string, bucketName string) (LogMonitor, error) {
	return &MinioLog{
		minioObjectName: objectName,
		bucketName:      bucketName,
		closed:          false,
	}, nil
}

func newMinioClient() (client *minio.Client, err error) {
	minioClient, err := minio.New(os.Getenv("MINIO_URL"), &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("MINIO_USERNAME"), os.Getenv("MINIO_PASSWORD"), ""),
		Secure: false,
	})
	if err != nil {
		fmt.Printf("create minio client error:%+v", err)
		return nil, err
	}
	return minioClient, nil
}
