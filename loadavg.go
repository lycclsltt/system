package system

import "strings"

//一分钟平均负载
func LoadAvg1(args string) string {
	content, err := GetFileContent("/proc/loadavg")
	if err != nil {
		return ""
	}
	fields := strings.Fields(content)
	return fields[0]
}
