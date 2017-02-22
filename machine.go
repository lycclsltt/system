package system

import (
	"fmt"
	"strconv"
	"strings"
)

//机型
func MachineProductName(args string) string {
	cmd := "dmidecode | grep 'Product Name' | uniq 2>/dev/null"
	output, err := Exec(cmd)
	if err != nil {
		return ""
	}
	if output == "" {
		return ""
	}
	lines := strings.Split(output, "\n")
	models := []string{}
	for _, line := range lines {
		if strings.Contains(line, "Product Name:") {
			model := strings.Replace(line, "Product Name:", "", -1)
			model = strings.TrimSpace(model)
			models = append(models, model)
		}
	}
	ret := strings.Join(models, "|")
	return ret
}

//操作系统版本
func OsVersion(args string) string {
	return ExecOutput("uname -sr")
}

//已运行时间(天)
func UpTime(args string) string {
	content, err := GetFileContent("/proc/uptime")
	if err != nil {
		return ""
	}
	strList := strings.Fields(content)
	upSeconds, err := strconv.ParseFloat(strList[0], 64)
	if err != nil {
		return ""
	}
	days := upSeconds / 3600 / 24
	return fmt.Sprintf("%.0f", days)
}
