package system

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func Exec(cmd string) (string, error) {
	command := exec.Command("sh", "-c", cmd)
	bytes, err := command.Output()
	return string(bytes), err
}

func Trim(str *string) {
	*str = strings.TrimSpace(*str)
	*str = strings.Replace(*str, "\n", "", -1)
}

func ExecOutput(cmd string) string {
	output, err := Exec(cmd)
	if err != nil {
		return ""
	}
	Trim(&output)
	return output
}

func FloatToString(f float64) string {
	if f == 0 {
		//0.00 => 0 减少带宽占用
		return "0"
	}
	return strconv.FormatFloat(f, 'f', 2, 64)
}

func GetFileContent(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
