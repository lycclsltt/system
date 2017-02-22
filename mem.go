package system

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
)

type Mem struct {
	Buffers      uint64
	Cached       uint64
	MemTotal     uint64
	MemFree      uint64
	MemUsed      uint64
	MemUsedRate  float64 //物理内存使用率
	SwapTotal    uint64
	SwapUsed     uint64
	SwapUsedRate float64 //交换内存使用率
	SwapFree     uint64
}

var WANT = map[string]struct{}{
	"Buffers:":   struct{}{},
	"Cached:":    struct{}{},
	"MemTotal:":  struct{}{},
	"MemFree:":   struct{}{},
	"SwapTotal:": struct{}{},
	"SwapFree:":  struct{}{},
}

func (this *Mem) Dump() {
	fmt.Printf("Buffers:%d, Cached:%d, MemTotal:%d, MemFree:%d, SwapTotal:%d, SwapUsed:%d, SwapFree:%d (kb)",
		this.Buffers,
		this.Cached,
		this.MemTotal,
		this.MemFree,
		this.SwapTotal,
		this.SwapUsed,
		this.SwapFree)
}

func (this *Mem) Collect() error {
	contents, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return err
	}
	reader := bufio.NewReader(bytes.NewBuffer(contents))

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		fields := strings.Fields(string(line))
		fieldName := fields[0]

		_, ok := WANT[fieldName]
		if ok && len(fields) == 3 {
			val, numerr := strconv.ParseUint(fields[1], 10, 64)
			if numerr != nil {
				continue
			}
			switch fieldName {
			case "Buffers:":
				this.Buffers = val
			case "Cached:":
				this.Cached = val
			case "MemTotal:":
				this.MemTotal = val
			case "MemFree:":
				this.MemFree = val
			case "SwapTotal:":
				this.SwapTotal = val
			case "SwapFree:":
				this.SwapFree = val
			}
		}
	}
	this.SwapUsed = this.SwapTotal - this.SwapFree
	//free + buffer + cached 才是可实际使用的内存
	this.MemFree = this.MemFree + this.Buffers + this.Cached
	this.MemUsed = this.MemTotal - this.MemFree
	if this.MemTotal > 0 {
		this.MemUsedRate = float64(this.MemUsed) / float64(this.MemTotal) * 100
	}
	if this.SwapTotal > 0 {
		this.SwapUsedRate = float64(this.SwapUsed) / float64(this.SwapTotal) * 100
	}
	return nil
}

//总内存大小(GB, 保留2位小数)
func (this *Mem) MemTotalGB(args string) string {
	memTotal := float64(this.MemTotal)
	return fmt.Sprintf("%.2f", memTotal/1024.0/1024.0)
}

//总内存大小(kb)
func (this *Mem) MemTotalFunc(args string) string {
	return strconv.FormatUint(this.MemTotal, 10)
}

//系统buffers大小(kb)
func (this *Mem) MemBuffer(args string) string {
	return strconv.FormatUint(this.Buffers, 10)
}

//系统cache大小(kb)
func (this *Mem) MemCachedFunc(args string) string {
	return strconv.FormatUint(this.Cached, 10)
}

//已使用物理内存(kb)
func (this *Mem) MemUsedFunc(args string) string {
	return strconv.FormatUint(this.MemUsed, 10)
}

//剩余物理内存(kb)
func (this *Mem) MemFreeFunc(args string) string {
	return strconv.FormatUint(this.MemFree, 10)
}

//剩余交换内存(kb)
func (this *Mem) SwapFreeFunc(args string) string {
	return strconv.FormatUint(this.SwapFree, 10)
}

//总交换内存(kb)
func (this *Mem) SwapTotalFunc(args string) string {
	return strconv.FormatUint(this.SwapTotal, 10)
}

//已使用交换内存(kb)
func (this *Mem) SwapUsedFunc(args string) string {
	return strconv.FormatUint(this.SwapUsed, 10)
}

//物理内存使用率
func (this *Mem) MemUsedRateFunc(args string) string {
	return FloatToString(this.MemUsedRate)
}

//交换内存使用率
func (this *Mem) SwapUsedRateFunc(args string) string {
	return FloatToString(this.SwapUsedRate)
}

//返回某进程的内存使用率
func MemUsedRateByProc(proc string) string {
	return ExecOutput("ps axuwww|grep " + proc + "|grep -v grep|awk 'BEGIN{sum=0}{sum+=$4 }END{print sum}'")
}
