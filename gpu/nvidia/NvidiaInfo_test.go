package nvidia

import (
	"fmt"
	"testing"
)

func TestTailLog(t *testing.T) {
	t.Log("start")
	out := "Fri Feb  2 10:26:30 2024\n+---------------------------------------------------------------------------------------+\n| NVIDIA-SMI 530.30.02              Driver Version: 530.30.02    CUDA Version: 12.1     |\n|-----------------------------------------+----------------------+----------------------+\n| GPU  Name                  Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |\n| Fan  Temp  Perf            Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |\n|                                         |                      |               MIG M. |\n|=========================================+======================+======================|\n|   0  Tesla T4                        Off| 00000000:00:0A.0 Off |                    0 |\n| N/A   65C    P0               43W /  70W|  10587MiB / 15360MiB |     96%      Default |\n|                                         |                      |                  N/A |"
	r, _ := GetInfoByString(out)
	fmt.Println(r)
}
