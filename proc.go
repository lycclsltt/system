package system

import (
	"fmt"
)

func ProcNumByKeyword(keyword string) string {
	cmd := fmt.Sprintf("ps auxww | grep %s | grep -v grep | wc -l", keyword)
	return ExecOutput(cmd)
}
