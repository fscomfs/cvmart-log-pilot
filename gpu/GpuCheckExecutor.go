package gpu

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
)

var checkGpuExec *exec.Cmd

func CheckGpu(appNum string) (int, error) {
	status := 0
	if checkGpuExec != nil {
		return 5, fmt.Errorf("CheckGpu started")
	}
	checkGpuExec = exec.Command("/usr/bin/nvidia_gpu_check", "-appNum", appNum)
	out, err := checkGpuExec.CombinedOutput()
	if err != nil {
		execError, ok := err.(*exec.ExitError)
		if ok {
			status = execError.ExitCode()
		}
	}
	r := bufio.NewReader(bytes.NewReader(out))
	for {
		res, e2 := r.ReadString('\n')
		if e2 != nil && e2 == io.EOF {
			break
		}
		log.Printf("nvidia_gpu_check exec result:%+v", res)
		if strings.Contains(res, "error") {
			return status, fmt.Errorf(res)
		}
	}

	checkGpuExec = nil
	return status, err
}

type CheckRes struct {
	Msg    string `json:"msg"`
	Status int    `json:"status"`
}
