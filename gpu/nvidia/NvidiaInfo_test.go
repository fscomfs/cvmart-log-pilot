package nvidia

import (
	"fmt"
	"testing"
)

func TestTailLog(t *testing.T) {
	t.Log("start")
	out := "Thu Jul  6 11:31:31 2023\n+-----------------------------------------------------------------------------+\n| NVIDIA-SMI 460.91.03    Driver Version: 460.91.03    CUDA Version: 11.2     |\n|-------------------------------+----------------------+----------------------+\n| GPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |\n| Fan  Temp  Perf  Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |\n|                               |                      |               MIG M. |\n|===============================+======================+======================|\n|   0  GeForce RTX 3090    Off  | 00000000:00:0B.0 Off |                  N/A |\n| 60%   57C    P2   110W / 350W |  19829MiB / 24268MiB |      0%      Default |\n|                               |                      |                  N/A |\n+-------------------------------+----------------------+----------------------+\n\n+-----------------------------------------------------------------------------+\n| Processes:                                                                  |\n|  GPU   GI   CI        PID   Type   Process name                  GPU Memory |\n|        ID   ID                                                   Usage      |\n|=============================================================================|\n|    0   N/A  N/A      5055      C   python                          19821MiB |\n+-----------------------------------------------------------------------------+\n"
	r, _ := GetInfoByString(out)
	fmt.Println(r)
}
