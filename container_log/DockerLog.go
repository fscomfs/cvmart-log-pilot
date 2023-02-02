package container_log

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	k8sApi "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"net"
	"net/http"
	"net/url"
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

type StatsEntry struct {
	Container        string
	Name             string
	ID               string
	CPUPercentage    float64
	Memory           float64 // On Windows this is the private working set
	MemoryLimit      float64 // Not used on Windows
	MemoryPercentage float64 // Not used on Windows
	NetworkRx        float64
	NetworkTx        float64
	BlockRead        float64
	BlockWrite       float64
	PidsCurrent      uint64 // Not used on Windows
	IsInvalid        bool
}

var dockerClient = make(map[string]*docker.Client)
var k8sClient *kubernetes.Clientset
var daemonOSType string

func init() {
	initK8sClient()
}
func initK8sClient() {
	config, err := rest.InClusterConfig()
	if err != nil {
		host, port, token := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT"), os.Getenv("KUBERNETES_TOKEN")
		config = &rest.Config{
			Host:        "https://" + net.JoinHostPort(host, port),
			BearerToken: token,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
		}
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
	if _, ok := dockerClient[dockerHost]; ok {
		c = dockerClient[dockerHost]
	} else {
		c = utils.NewDockerClient(dockerHost)
		if c != nil {
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

func (d *DockerLog) Start(ctx context.Context, def *ConnectDef) error {
	var containerId string
	var dockerHost string
	if def.LogClaims.ContainerId == "" && def.LogClaims.PodLabel != "" && d.k8sClient != nil {
		for {
			if d.closed {
				log.Printf("DockerLog closed id=%+v", def.Id)
				return nil
			}
			podList, err := k8sClient.CoreV1().Pods("default").List(context.Background(), v1.ListOptions{
				Watch:         false,
				LabelSelector: "app=" + def.LogClaims.PodLabel,
			})

			if err != nil || len(podList.Items) == 0 {
				def.write(logMessage([]byte("\rWait for task initiation...")))
				time.Sleep(5 * time.Second)
				continue
			}
			pod := podList.Items[0]
			count := len(pod.Status.InitContainerStatuses) + len(pod.Status.ContainerStatuses)
			if pod.Status.Phase == "Pending" {
				process := GetPodProcess(pod.Status)
				if process == count {
					def.write(logMessage([]byte("\rtask init:" + fmt.Sprint(process) + string("/") + fmt.Sprint(count) + "\n")))
				} else {
					def.write(logMessage([]byte("\rtask init:" + fmt.Sprint(process) + string("/") + fmt.Sprint(count))))
				}
			}
			if pod.Status.Phase == "Running" || pod.Status.Phase == "Succeeded" {
				if len(pod.Status.ContainerStatuses) > 0 {
					process := GetPodProcess(pod.Status)
					if process == count {
						def.write(logMessage([]byte("\rtask init:" + fmt.Sprint(process) + string("/") + fmt.Sprint(count) + "\n")))
					} else {
						def.write(logMessage([]byte("\rtask init:" + fmt.Sprint(process) + string("/") + fmt.Sprint(count))))
					}
					containerId = strings.TrimPrefix(pod.Status.ContainerStatuses[0].ContainerID, "docker://")
					dockerHost = pod.Status.HostIP + ":2375"
					break
				}
			}
			if pod.Status.Phase == "Failed" || pod.Status.Phase == "Unknown" {
				log.Printf("pod status:%+v", pod.Status.Message)
				def.write(logMessage([]byte("task run fail:" + pod.Status.Reason + "\n")))
				return nil
			}
			time.Sleep(5 * time.Second)
		}
	} else {
		containerId = def.LogClaims.ContainerId
		dockerHost = def.LogClaims.Host + ":" + def.LogClaims.Port
	}
	log.Printf("start tail container log:%+v", def.LogClaims)
	if d.client == nil {
		d.dockerHost = dockerHost
		d.client = GetDockerClient(d.dockerHost)
	}
	tail := "10000"
	if def.LogClaims.Tail != "" {
		tail = def.LogClaims.Tail
	}
	go func() {
		for {
			h, _ := containerGpuInfo(ctx, strings.Split(d.dockerHost, ":")[0]+fmt.Sprintf(":%d", utils.ServerPort), containerId, func(res []byte) {
				def.write(gpuMessage(res))
			})
			if !h {
				break
			}
			time.Sleep(5 * time.Second)
		}
	}()
	go func() {
		for {
			h, _ := containerResourceInfo(ctx, d.client, containerId, func(res []byte) {
				def.write(resourceMessage(res))
			})
			if !h {
				break
			}
			time.Sleep(5 * time.Second)
		}
	}()

	reader, err := d.client.ContainerLogs(ctx, containerId, types.ContainerLogsOptions{
		Follow:     true,
		ShowStderr: true,
		ShowStdout: true,
		Tail:       tail,
		Timestamps: false,
		Details:    true,
	})
	if err != nil {
		return err
	}
	defer reader.Close()
	defer def.onClose()
	r := bufio.NewReader(reader)
	var out = ioutil.Discard
	StdCopy(out, r, def.write)
	def.write(logMessage([]byte("log end")))
	log.Printf("tail log process end id=%+v", def.Id)
	return nil
}

func (d *DockerLog) Close() error {
	d.closed = true
	return nil
}

func containerGpuInfo(ctx context.Context, host string, containerID string, handler func(res []byte)) (bool, error) {
	u := url.URL{Scheme: "ws", Host: host, Path: utils.API_CONTAINERGPUINFO}
	c, r, err := websocket.DefaultDialer.Dial(u.String()+"?containerID="+containerID, nil)
	if r.StatusCode == http.StatusNoContent {
		return false, fmt.Errorf("not content")
	}
	if err != nil {
		return true, err
	}
	defer c.Close()
	for {
		select {
		case <-ctx.Done():
			log.Printf("-----------gpu end----------------------")
			return false, nil
		default:
			_, content, err := c.ReadMessage()
			if err != nil {
				return true, err
			}
			handler(content)
		}

	}
	return true, nil
}

func containerResourceInfo(ctx context.Context, client *docker.Client, containerID string, handlerCallback func(res []byte)) (bool, error) {
	var (
		previousCPU    uint64
		previousSystem uint64
	)
	res, err := client.ContainerStats(ctx, containerID, true)
	defer res.Body.Close()
	if err != nil {
		return false, err
	}
	dec := json.NewDecoder(res.Body)

	for {
		select {
		case <-ctx.Done():
			log.Printf("-----------resource----------------------")
			return false, nil
		default:
			var (
				v                      *types.StatsJSON
				memPercent, cpuPercent float64
				blkRead, blkWrite      uint64 // Only used on Linux
				mem, memLimit          float64
				pidsStatsCurrent       uint64
			)
			if err := dec.Decode(&v); err != nil {
				dec = json.NewDecoder(io.MultiReader(dec.Buffered(), res.Body))
				if err == io.EOF {
					break
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}
			daemonOSType := res.OSType
			if daemonOSType != "windows" {
				previousCPU = v.PreCPUStats.CPUUsage.TotalUsage
				previousSystem = v.PreCPUStats.SystemUsage
				cpuPercent = calculateCPUPercentUnix(previousCPU, previousSystem, v)
				blkRead, blkWrite = calculateBlockIO(v.BlkioStats)
				mem = calculateMemUsageUnixNoCache(v.MemoryStats)
				memLimit = float64(v.MemoryStats.Limit)
				memPercent = calculateMemPercentUnixNoCache(memLimit, mem)
				pidsStatsCurrent = v.PidsStats.Current
			} else {
				cpuPercent = calculateCPUPercentWindows(v)
				blkRead = v.StorageStats.ReadSizeBytes
				blkWrite = v.StorageStats.WriteSizeBytes
				mem = float64(v.MemoryStats.PrivateWorkingSet)
			}
			netRx, netTx := calculateNetwork(v.Networks)
			stats := StatsEntry{
				Name:             v.Name,
				ID:               v.ID,
				CPUPercentage:    cpuPercent,
				Memory:           mem,
				MemoryPercentage: memPercent,
				MemoryLimit:      memLimit,
				NetworkRx:        netRx,
				NetworkTx:        netTx,
				BlockRead:        float64(blkRead),
				BlockWrite:       float64(blkWrite),
				PidsCurrent:      pidsStatsCurrent,
			}
			if res, err := json.Marshal(stats); err == nil {
				handlerCallback(res)
			}
		}
	}
}

func calculateCPUPercentUnix(previousCPU, previousSystem uint64, v *types.StatsJSON) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.CPUStats.SystemUsage) - float64(previousSystem)
		onlineCPUs  = float64(v.CPUStats.OnlineCPUs)
	)

	if onlineCPUs == 0.0 {
		onlineCPUs = float64(len(v.CPUStats.CPUUsage.PercpuUsage))
	}
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}
	return cpuPercent
}

func calculateBlockIO(blkio types.BlkioStats) (uint64, uint64) {
	var blkRead, blkWrite uint64
	for _, bioEntry := range blkio.IoServiceBytesRecursive {
		if len(bioEntry.Op) == 0 {
			continue
		}
		switch bioEntry.Op[0] {
		case 'r', 'R':
			blkRead = blkRead + bioEntry.Value
		case 'w', 'W':
			blkWrite = blkWrite + bioEntry.Value
		}
	}
	return blkRead, blkWrite
}

func calculateNetwork(network map[string]types.NetworkStats) (float64, float64) {
	var rx, tx float64

	for _, v := range network {
		rx += float64(v.RxBytes)
		tx += float64(v.TxBytes)
	}
	return rx, tx
}

// calculateMemUsageUnixNoCache calculate memory usage of the container.
// Cache is intentionally excluded to avoid misinterpretation of the output.
//
// On cgroup v1 host, the result is `mem.Usage - mem.Stats["total_inactive_file"]` .
// On cgroup v2 host, the result is `mem.Usage - mem.Stats["inactive_file"] `.
//
// This definition is consistent with cadvisor and containerd/CRI.
// * https://github.com/google/cadvisor/commit/307d1b1cb320fef66fab02db749f07a459245451
// * https://github.com/containerd/cri/commit/6b8846cdf8b8c98c1d965313d66bc8489166059a
//
// On Docker 19.03 and older, the result was `mem.Usage - mem.Stats["cache"]`.
// See https://github.com/moby/moby/issues/40727 for the background.
func calculateMemUsageUnixNoCache(mem types.MemoryStats) float64 {
	// cgroup v1
	if v, isCgroup1 := mem.Stats["total_inactive_file"]; isCgroup1 && v < mem.Usage {
		return float64(mem.Usage - v)
	}
	// cgroup v2
	if v := mem.Stats["inactive_file"]; v < mem.Usage {
		return float64(mem.Usage - v)
	}
	return float64(mem.Usage)
}

func calculateMemPercentUnixNoCache(limit float64, usedNoCache float64) float64 {
	// MemoryStats.Limit will never be 0 unless the container is not running and we haven't
	// got any data from cgroup
	if limit != 0 {
		return usedNoCache / limit * 100.0
	}
	return 0
}

func calculateCPUPercentWindows(v *types.StatsJSON) float64 {
	// Max number of 100ns intervals between the previous time read and now
	possIntervals := uint64(v.Read.Sub(v.PreRead).Nanoseconds()) // Start with number of ns intervals
	possIntervals /= 100                                         // Convert to number of 100ns intervals
	possIntervals *= uint64(v.NumProcs)                          // Multiple by the number of processors

	// Intervals used
	intervalsUsed := v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage

	// Percentage avoiding divide-by-zero
	if possIntervals > 0 {
		return float64(intervalsUsed) / float64(possIntervals) * 100.0
	}
	return 0.00
}
