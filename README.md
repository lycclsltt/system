## 采集linux指标, 可用于监控
### 包括:
### cpu
### 内存
### io状态
### 磁盘占用
### 负载
### 内外网卡流入、流出流量

## 使用
```go
package main

import(
	"github.com/lycclsltt/system"
)

func main() {
    mem := &system.Mem{}
	mem.Collect()
	println("used:", mem.MemUsedFunc(""), "kb")
	println("free:", mem.MemFreeFunc(""), "kb")
}
```
