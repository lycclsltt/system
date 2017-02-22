package system

//dmesg出现error行数
func DmesgErrCount(args string) string {
	return ExecOutput("dmesg|grep error|wc -l")
}
