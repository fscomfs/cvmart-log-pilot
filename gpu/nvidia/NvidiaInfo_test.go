package nvidia

import (
	"fmt"
	"testing"
)

func TestTailLog(t *testing.T) {
	t.Log("start")
	out := "Thu Feb  1 14:58:24 2024\n+---------------------------------------------------------------------------------------+\n| NVIDIA-SMI 530.30.02              Driver Version: 530.30.02    CUDA Version: 12.1     |\n|-----------------------------------------+----------------------+----------------------+\n| GPU  Name                  Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |\n| Fan  Temp  Perf            Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |\n|                                         |                      |               MIG M. |\n|=========================================+======================+======================|\n|   0  NVIDIA GeForce RTX 3090         Off| 00000000:00:0A.0 Off |                  N/A |\n| 30%   45C    P2              105W / 350W|  10902MiB / 24576MiB |      0%      Default |\n|                                         |                      |                  N/A |\n+-----------------------------------------+----------------------+----------------------+\n\n+---------------------------------------------------------------------------------------+\n| Processes:                                                                            |\n|  GPU   GI   CI        PID   Type   Process name                            GPU Memory |\n|        ID   ID                                                             Usage      |\n|=======================================================================================|\n|    0   N/A  N/A     24437      C   python                                    10886MiB |\n+---------------------------------------------------------------------------------------+"
	r, _ := GetInfoByString(out)
	fmt.Println(r)
}
