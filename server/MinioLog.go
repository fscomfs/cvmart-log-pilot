package server

import (
	"bufio"
	"context"
	"fmt"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"os"
)

var minioClient *minio.Client

func init() {
	var err error
	minioClient, err = newMinioClient()
	if err != nil {
		//panic(err.Error())
	}
}

type MinioLog struct {
	minioObjectName string `json:"minio_object_name"`
	bucketName      string `json:"bucket_name"`
	closed          bool   `json:"closed"`
}

func (m *MinioLog) Start(def *ConnectDef) error {
	object, err := minioClient.GetObject(context.Background(), m.bucketName, m.minioObjectName, minio.GetObjectOptions{})
	if err != nil {
		def.WriteMsg <- []byte(err.Error() + "\n")
		m.closed = true
		return err
	}

	r := bufio.NewReader(object)
	defer object.Close()
	for {
		line, e := r.ReadBytes('\n')
		if e != nil {
			def.WriteMsg <- []byte(e.Error() + "\n")
			return e
		}
		if m.closed {
			return nil
		}
		def.WriteMsg <- line
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
