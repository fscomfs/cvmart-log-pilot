package server

import (
	"bufio"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"io/ioutil"
	k8sApi "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type DockerLog struct {
	dockerHost string
	client     *docker.Client
	k8sClient  *kubernetes.Clientset
	closed     bool
	podLabel   string
}

var dockerClient = make(map[string]*docker.Client)
var k8sClient *kubernetes.Clientset

func init() {
	initK8sClient()
}
func initK8sClient() {
	host, port, token := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT"), os.Getenv("KUBERNETES_TOKEN")
	config := &rest.Config{
		Host:        "https://" + net.JoinHostPort(host, port),
		BearerToken: token,
		//	BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	k8sClient = clientset
}

func NewDockerLog(dockerHost string) (LogMonitor, error) {
	c := GetDockerClient(dockerHost)
	return &DockerLog{
		dockerHost: dockerHost,
		client:     c,
		k8sClient:  k8sClient,
		closed:     false,
	}, nil
}

func GetDockerClient(dockerHost string) (client *docker.Client) {
	var c *docker.Client
	var err error
	if _, ok := dockerClient[dockerHost]; ok {
		c = dockerClient[dockerHost]
	} else {
		if dockerHost != "" {
			if !strings.HasPrefix(dockerHost, "http://") {
				dockerHost = "http://" + dockerHost
			}
			c, err = docker.NewClient(dockerHost, "", nil, nil)
			if err != nil {
				return nil
			}
			dockerClient[dockerHost] = c
		}
	}
	return c
}

func GetPodProcess(status k8sApi.PodStatus) int {
	process := 0
	for _, v := range status.InitContainerStatuses {
		if v.Ready {
			process += 1
		}
	}
	for _, v := range status.ContainerStatuses {
		if v.Ready {
			process += 1
		}
	}
	return process
}

func (l *DockerLog) Start(def *ConnectDef) error {
	ctx := context.Background()
	var containerId string
	var dockerHost string
	if def.LogClaims.ContainerId == "" && def.LogClaims.PodLabel != "" && l.k8sClient != nil {
		for {
			if l.closed {
				return nil
			}
			podList, err := k8sClient.CoreV1().Pods("default").List(ctx, v1.ListOptions{
				Watch:         false,
				LabelSelector: "app=" + def.LogClaims.PodLabel,
			})
			if err != nil || len(podList.Items) == 0 {
				continue
			}
			pod := podList.Items[0]
			count := len(pod.Status.InitContainerStatuses) + len(pod.Status.ContainerStatuses)
			if pod.Status.Phase == "Pending" {
				process := GetPodProcess(pod.Status)
				if process == count {
					def.WriteMsg <- []byte("\rtask init:" + fmt.Sprint(process) + string("/") + fmt.Sprint(count) + "\n")
				} else {
					def.WriteMsg <- []byte("\rtask init:" + fmt.Sprint(process) + string("/") + fmt.Sprint(count))
				}
			}
			if pod.Status.Phase == "Running" || pod.Status.Phase == "Succeeded" {
				if len(pod.Status.ContainerStatuses) > 0 {
					process := GetPodProcess(pod.Status)
					if process == count {
						def.WriteMsg <- []byte("\rtask init:" + fmt.Sprint(process) + string("/") + fmt.Sprint(count) + "\n")
					} else {
						def.WriteMsg <- []byte("\rtask init:" + fmt.Sprint(process) + string("/") + fmt.Sprint(count))
					}
					containerId = strings.TrimPrefix(pod.Status.ContainerStatuses[0].ContainerID, "docker://")
					dockerHost = pod.Status.HostIP + ":2375"
					break
				}
			}
			if pod.Status.Phase == "Failed" || pod.Status.Phase == "Unknown" {
				log.Printf("pod status:%+v", pod.Status.Message)
				def.WriteMsg <- []byte("task run fail:" + pod.Status.Reason + "\n")
				return nil
			}
			time.Sleep(5 * time.Second)
		}
	}
	log.Printf("start tail container log:%+v", def.LogClaims)
	if l.client == nil {
		l.dockerHost = dockerHost
		l.client = GetDockerClient(l.dockerHost)
	}
	reader, err := l.client.ContainerLogs(ctx, containerId, types.ContainerLogsOptions{
		Follow:     true,
		ShowStderr: false,
		ShowStdout: true,
		Tail:       "1000",
		Timestamps: false,
		Details:    false,
	})
	if err != nil {
		return err
	}
	defer reader.Close()
	r := bufio.NewReader(reader)
	var out = ioutil.Discard
	StdCopy(out, def.WriteMsg, r)

	def.WriteMsg <- []byte("log end")
	//for {
	//	var line []byte
	//	var out = ioutil.Discard
	//	StdCopy(out, nil, r)
	//	ioutil.Discard.Write(line)
	//	//line, err := r.ReadBytes('\n')
	//	//if err != nil {
	//	//	return err
	//	//}
	//	if l.closed {
	//		return nil
	//	}
	//	def.WriteMsg <- line
	//}

	return nil
}

func (l *DockerLog) Close() error {
	l.closed = true
	return nil
}
