package container_log

import (
	"context"
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"log"
	"os"
	"regexp"
	"testing"
	"time"
)

func init1() {
	os.Setenv("JWT_SEC", "111")

}
func TestTailLog(t *testing.T) {
	init1()
	l, err := NewDockerLog("http://localhost:2375")
	if err != nil {

	}
	logParam := LogParam{
		ContainerId: "4d07f21edd9c",
		Host:        "localhost",
		Operator:    "log",
		Tail:        "50000",
	}
	GeneratorToken(logParam, 100)
	c := &ConnectDef{
		Connect:  nil,
		LogParam: &logParam,
		WriteMsg: make(chan []byte),
	}
	go func() {
		for {
			select {
			case s := <-c.WriteMsg:
				fmt.Print(string(s))
			}
		}
	}()
	l.Start(context.Background(), c)

}
func init2() {
	os.Setenv("KUBERNETES_SERVICE_HOST", "192.168.1.131")
	os.Setenv("KUBERNETES_SERVICE_PORT", "6443")
	os.Setenv("KUBERNETES_TOKEN", "eyJhbGciOiJSUzI1NiIsImtpZCI6Ii1EQkxmRVlFMnFBNm1xcHk3U2NhSm0xUGhaZnlsT2dZeFNIZUFRZzBLU0UifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJrdWJlLXN5c3RlbSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJhZG1pbi10b2tlbi14bjh0cCIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50Lm5hbWUiOiJhZG1pbiIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50LnVpZCI6IjYyOTQ0YmQwLTNhN2MtNDNlYy05MTYyLTc3ZGNjYTEwMDU3NyIsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDprdWJlLXN5c3RlbTphZG1pbiJ9.JCyjrKbVMVis28DIAjp1L9BwlqT3XXGrTHH_oUN_4Xu6gcOP2GOokg9S66CXZR7CSPTtTWbpRFHu3KyoISQFl5TxatDGrHvEjbMtcugwHBTW6yrfxJs_woN4QphlFq5wBzmwcpvC1MXuj3VTIRvabnivfL3wa2qw3iccP8eYSPpaySVKChu60WW_oYMrvVOL3PG01DlWY2PuVS6-uHliCal5_lY22VWKo8AROpoe8tWVa5YEeY45LEe9bsK-WXqY9OweN3PLOGELpjAeY5wc5GJCsACm9Jvv43CfuGopz7rKD005dlfojY4GvF9IVnEMSXFtv0ZtDRSMwPqECubsnQ")
}
func TestLog(t *testing.T) {
	init2()
	cLog, err := NewDockerLog("")
	if err != nil {

	}
	utils.InitK8sClient()
	c := &ConnectDef{
		Id: "111",
		LogParam: &LogParam{
			PodLabel: "log-test",
		},
		WriteMsg: make(chan []byte),
	}
	connectHub.connects["111"] = c
	defer cLog.Close()
	go func() {
		for {
			select {
			case l := <-c.WriteMsg:
				fmt.Println(l)
			}
		}
	}()
	cLog.Start(context.Background(), c)

}

func TestR(t *testing.T) {
	reg, _ := regexp.Compile(`172\.16\..`)
	fmt.Printf("d:%+v", reg.MatchString("172.16.1115"))
}

func TestGoContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Print("end 1")
				return
			default:
				for {
					log.Print("exec 1")
					time.Sleep(time.Second * 1)
				}
			}
		}
	}()
	time.Sleep(time.Second * 2)
	cancel()
	time.Sleep(time.Second * 3)

}
