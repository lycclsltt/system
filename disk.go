package system

import (
	"fmt"
	"strconv"
	"strings"
)

type FileSystem struct {
	FsName   string  //文件系统
	Total    uint64  //总空间(kb)
	Used     uint64  //已用空间(kb)
	Free     uint64  //可用空间(kb)
	UsedRate float64 //已用(百分比)
	Mount    string  //挂载点
}

type Disk struct {
	UsedRateSet  []string              //磁盘所有分区使用率集合
	FsMap        map[string]FileSystem //挂载点=>使用情况
	Total        uint64                //所有挂载分区总空间(kb)
	Used         uint64                //所有挂载分区总已用空间(kb)
	Free         uint64                //所有挂载分区总可用空间(kb)
	UsedRate     float64               //所有挂载分区总使用率
	MaxUseRate   float64               //所有挂载分区最大使用率
	MaxUseRateFs string                //使用率最大的分区
}

var mountedSet = []string{
	"/disk",
	"/",
	"/home",
	"/tmp",
	"/var",
	"/dev/shm",
	"/boot",
	"/usr",
	"/data",
	"/data0",
	"/data1",
	"/data2",
	"/data3",
	"/data4",
	"/data5",
	"/data6",
	"/data7",
	"/data8",
	"/data9",
	"/data10",
	"/data11",
	"/data12",
	"/data00",
	"/data01",
	"/data02",
	"/data03",
	"/data04",
	"/data05",
	"/data06",
	"/data07",
	"/data08",
	"/data09",
}

func (this *Disk) Collect() error {
	//注意，这里一定要加P, 否则当Filesystem 太长时，无法显示为一行, 采集数据会不全
	cmd := "df -lP"
	output, err := Exec(cmd)
	if err != nil {
		return err
	}
	lines := strings.Split(output, "\n")
	this.FsMap = map[string]FileSystem{}
	this.UsedRateSet = []string{}
	var (
		maxUseRate   float64
		maxUseRateFs string
	)
	for row, line := range lines {
		if row == 0 {
			continue
		}
		fileds := strings.Fields(line)
		if len(fileds) != 6 {
			continue
		}
		fsName := fileds[0]
		total, _ := strconv.ParseUint(fileds[1], 10, 64)
		used, _ := strconv.ParseUint(fileds[2], 10, 64)
		free, _ := strconv.ParseUint(fileds[3], 10, 64)
		strUsedRate := strings.Replace(fileds[4], "%", "", -1)
		usedRate, _ := strconv.ParseFloat(strUsedRate, 64)
		mount := fileds[5]
		//挂载分区使用情况
		fs := FileSystem{}
		fs.FsName = fsName
		fs.Total = total
		fs.Used = used
		fs.Free = free
		fs.UsedRate = usedRate
		fs.Mount = mount
		this.FsMap[mount] = fs
		//更新所有挂载分区使用情况
		this.Total += total
		this.Used += used
		this.Free += free
		//磁盘所有分区使用率集合
		this.UsedRateSet = append(this.UsedRateSet, fsName+"="+mount+"="+strUsedRate)
		if fs.FsName != "devfs" && fs.UsedRate > maxUseRate {
			maxUseRate = fs.UsedRate
			maxUseRateFs = fs.Mount
		}
	}

	//所有挂载分区总使用率
	if this.Used+this.Free > 0 {
		this.UsedRate = float64(this.Used) / float64(this.Used+this.Free) * 100
	}

	this.MaxUseRate = maxUseRate
	this.MaxUseRateFs = maxUseRateFs

	return nil
}

func (this *Disk) Dump() {
	for _, fs := range this.FsMap {
		fmt.Printf("FsName:%s, Total:%d, Used:%d, Free:%d, UsedRate:%f, Mount:%s\n",
			fs.FsName,
			fs.Total,
			fs.Used,
			fs.Free,
			fs.UsedRate,
			fs.Mount)
	}
}

//分区使用率
func (this *Disk) MountUsedRate(mount string) string {
	fs, exists := this.FsMap[mount]
	if !exists {
		return ""
	}
	return FloatToString(fs.UsedRate)
}

//所有挂载分区总使用率
func (this *Disk) DiskUsedRate(args string) string {
	return FloatToString(this.UsedRate)
}

//机器物理磁盘信息
func DiskModel(args string) string {
	cmd := "fdisk -l 2>/dev/null | grep 'Disk /dev/'"
	output, err := Exec(cmd)
	if err != nil {
		return ""
	}
	lines := strings.Split(output, "\n")
	models := []string{}
	for _, line := range lines {
		//去掉字节大小，只保留一个单位(GB或M)
		sizes := strings.Split(line, ",")
		if len(sizes) != 2 {
			continue
		}
		fields := strings.Fields(sizes[0])
		if len(fields) != 4 {
			continue
		}
		model := strings.Replace(fields[1], ":", "", -1)
		capacity := fields[2]
		models = append(models, model+"|"+capacity)
	}
	ret := strings.Join(models, "$") + "$"
	return ret
}

//磁盘所有分区使用率集合
func (this *Disk) DiskUsedRateSet(args string) string {
	return strings.Join(this.UsedRateSet, "$") + "$"
}

// /home分区剩余空间(MB)
func (this *Disk) HomeFree(args string) string {
	fs, exists := this.FsMap["/home"]
	if !exists {
		return ""
	}
	//MB
	free := float64(fs.Free) / float64(1024)
	return FloatToString(free)
}

// /home分区总空间(MB)
func (this *Disk) HomeTotal(args string) string {
	fs, exists := this.FsMap["/home"]
	if !exists {
		return ""
	}
	//MB
	total := float64(fs.Total) / float64(1024)
	return FloatToString(total)
}

// /home分区空间使用率
func (this *Disk) HomeUsedRate(args string) string {
	fs, exists := this.FsMap["/home"]
	if !exists {
		return ""
	}
	return FloatToString(fs.UsedRate)
}

//根分区剩余空间(MB)
func (this *Disk) RootFree(args string) string {
	fs, exists := this.FsMap["/"]
	if !exists {
		return ""
	}
	free := float64(fs.Free) / float64(1024)
	return FloatToString(free)
}

//根分区总空间(MB)
func (this *Disk) RootTotal(args string) string {
	fs, exists := this.FsMap["/"]
	if !exists {
		return ""
	}
	total := float64(fs.Total) / float64(1024)
	return FloatToString(total)
}

//根分区空间使用率
func (this *Disk) RootUsedRate(args string) string {
	fs, exists := this.FsMap["/"]
	if !exists {
		return ""
	}
	return FloatToString(fs.UsedRate)
}

// /tmp分区使用率
func (this *Disk) TmpUsedRate(args string) string {
	fs, exists := this.FsMap["/tmp"]
	if !exists {
		return ""
	}
	return FloatToString(fs.UsedRate)
}

// /usr分区使用率
func (this *Disk) UsrUsedRate(args string) string {
	fs, exists := this.FsMap["/usr"]
	if !exists {
		return ""
	}
	return FloatToString(fs.UsedRate)
}

// /var分区使用率
func (this *Disk) VarUsedRate(args string) string {
	fs, exists := this.FsMap["/var"]
	if !exists {
		return ""
	}
	return FloatToString(fs.UsedRate)
}

//磁盘所有分区最大使用率
func (this *Disk) MaxUsedRateFsFunc(args string) string {
	return FloatToString(this.MaxUseRate) + "," + this.MaxUseRateFs
}

//返回某个目录的大小(MB)
func DiskUsedByDir(dir string) string {
	return ExecOutput("du -sm " + dir + "|awk '{print $1}'")
}
