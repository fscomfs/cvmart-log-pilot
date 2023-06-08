package bmctl

import (
	"fmt"
	"testing"
)

func TestBMCTL_GET_DEV_CNT(t *testing.T) {
	w := BMCTL_GET_SMI_ATTR()
	fmt.Sprint(w.Exec(0, nil))
}
