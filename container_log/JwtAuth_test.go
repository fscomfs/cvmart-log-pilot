package container_log

import (
	"github.com/fscomfs/cvmart-log-pilot/config"
	"log"
	"testing"
)

func TestAuth(t *testing.T) {
	Auth("")
}

func TestGeneratorToken(t *testing.T) {
	claims := LogParam{
		Host:        "localhost",
		Operator:    OPERATOR_LOG,
		Tail:        "50000",
		ContainerId: "a2e31b25d36e",
	}
	config.ParseFromFile("/etc/cvmart/daemon-config.json")
	auth, err := GeneratorToken(claims, 100)
	if err != nil {
		t.Fatalf("GeneratorToken err:%+v", err)
	}
	//time.Sleep(3 * time.Second)
	//s, err := Auth("eyJhbGciOiJIUzI1NiJ9.eyJwb2RMYWJlbCI6ImxvZy10ZXN0IiwiZXhwIjoxNjcwOTA1MDcxLCJpYXQiOjE2NzA5MDQ5OTksIm9wZXJhdG9yIjoibG9nIn0.OCe-8rJ3yWo3FzE3eKZ2exWvFflB7_7SPb6YS7fUZ8s")
	//if err != nil {
	//	t.Errorf("err:%+v", err)
	//} else {
	//	t.Logf("loginInfo:%+v", *s)
	//}

	log.Print(auth)
}
