package system

import (
	"bufio"
	"errors"
	"io"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Partition struct {
	Name string
	//以下从系统启动后累加
	Rio      int64 //成功读的次数
	Rmerge   int64 //合并读的次数(为了效率内核会合并相邻的读或写)
	Rsect    int64 //成功读扇区的次数
	Relapsed int64 //所有读花费的时间(毫秒)
	Wio      int64 //成功写的次数
	Wmerge   int64 //合并写的次数
	Wsect    int64 //成功写扇区的次数
	Welapsed int64 //所有写花费的时间(毫秒)
	Elapsed  int64 //I/O操作花费的毫秒数
	Aveq     int64 //权重值，当有I/O操作时，这个值就增加

	//计算得出
	RmergePerSecond float64 //平均每秒合并读次数
	WmergePerSecond float64 //平均每秒合并写次数
	RioPerSecond    float64 //平均每秒读完成次数
	WioPerSecond    float64 //平均每秒写完成次数
	RsectPerSecond  float64 //平均每秒读扇区次数
	WsectPerSecond  float64 //平均每秒写扇区次数
	ReqSz           float64 //平均I/O大小
	ServeElapsed    float64 //平均I/O服务时间(ms)
	AwaitElapsed    float64 //平均io等待时间(ms)
	QueueSz         float64 //平均I/O队列长度
	ReqRate         float64 //平均I/O操作百分比
	RkbPerSecond    float64 //每秒读kb数
	WkbPerSecond    float64 //每秒写kb数

	Last int64 //上次采集时间
}

type DiskIO struct {
	PartiMap        map[string]*Partition //分区名称=>分区
	PartiNames      []string              //分区名称集合
	QueueSzAvg      float64               //所有分区平均I/O队列长度
	ReqSzAvg        float64               //所有分区平均I/O大小
	ServeAvg        float64               //所有分区平均I/O服务时间
	AwaitAvg        float64               //所有分区平均I/O等待时间
	RkbAvg          float64               //所有分区平均每秒读kb数
	WkbAvg          float64               //所有分区平均每秒写kb数
	RmergeAvg       float64               //所有分区平均每秒merge读次数
	WmergeAvg       float64               //所有分区平均每秒merge写次数
	RioAvg          float64               //所有分区平均每秒读完成次数
	WioAvg          float64               //所有分区平均每秒写完成次数
	RsectAvg        float64               //所有分区平均每秒读扇区次数
	WsectAvg        float64               //所有分区平均每秒写扇区次数
	ReqRateAvg      float64               //所有分区平均I/O操作百分比
	MaxReqRate      float64               //磁盘I/O最大使用率
	MaxReqRateParti string                //磁盘I/O使用率最大的分区名
}

func (this *DiskIO) Collect() error {
	err := this.InitPartitions()
	if err != nil {
		return err
	}
	err = this.InitIoStat()
	if err != nil {
		return err
	}
	return nil
}

func (this *DiskIO) Dump() {
	for name, _ := range this.PartiMap {
		fmt.Println("partition name:" + name)
	}
}

//读/proc/partitions, 采集分区名称
func (this *DiskIO) InitPartitions() error {
	f, err := os.Open("/proc/partitions")
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	row := 0
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		row++
		if row == 1 {
			continue
		}
		fileds := strings.Fields(line)
		if len(fileds) != 4 {
			continue
		}
		name := fileds[3]
		_, exists := this.PartiMap[name]
		if !exists {
			this.PartiMap[name] = &Partition{Name: name}
			this.PartiNames = append(this.PartiNames, name)
		}
	}

	return nil
}

//读/proc/diskstats, 采集io状态
func (this *DiskIO) InitIoStat() error {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)

	var (
		QueueSzSum float64 //平均I/O队列长度之和
		ReqSzSum   float64 //平均I/O大小之和
		ServeSum   float64 //平均I/O服务时间之和
		AwaitSum   float64 //平均I/O等待时间之和
		RkbSum     float64 //平均每秒读kb数之和
		WkbSum     float64 //平均每秒写kb数之和
		RmergeSum  float64 //平均每秒merge读次数之和
		WmergeSum  float64 //平均每秒merge写次数之和
		RioSum     float64 //平均每秒读次数之和
		WioSum     float64 //平均每秒写次数之和
		RsectSum   float64 //平均每秒读扇区次数之和
		WsectSum   float64 //平均每秒写扇区次数之和
		ReqRateSum float64 //平均I/O操作百分比之和
	)

	var (
		maxReqRate      float64 //磁盘I/O最大使用率
		maxReqRateParti string  ////磁盘I/O使用率最大的分区名
		//maxReqRateName string  //磁盘I/O使用率最大的分区名
	)

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fields := strings.Fields(line)
		if len(fields) != 14 {
			continue
		}
		partiName := fields[2]
		parti, exists := this.PartiMap[partiName]
		if !exists {
			continue
		}

		rio, _ := strconv.ParseInt(fields[3], 10, 64)
		rmerge, _ := strconv.ParseInt(fields[4], 10, 64)
		rsect, _ := strconv.ParseInt(fields[5], 10, 64)
		relapsed, _ := strconv.ParseInt(fields[6], 10, 64)
		wio, _ := strconv.ParseInt(fields[7], 10, 64)
		wmerge, _ := strconv.ParseInt(fields[8], 10, 64)
		wsect, _ := strconv.ParseInt(fields[9], 10, 64)
		welapsed, _ := strconv.ParseInt(fields[10], 10, 64)
		elapsed, _ := strconv.ParseInt(fields[12], 10, 64)
		aveq, _ := strconv.ParseInt(fields[13], 10, 64)
		now := time.Now().Unix()

		if parti.Last <= 0 {
			//第一次采集，还没产生时间差，不计算
		} else {
			difftime := float64(now - parti.Last)
			if difftime > 0 {
				parti.RmergePerSecond = float64(rmerge-parti.Rmerge) / difftime            //平均每秒合并读次数
				parti.WmergePerSecond = float64(wmerge-parti.Wmerge) / difftime            //平均每秒合并写次数
				parti.RioPerSecond = float64(rio-parti.Rio) / difftime                     //平均每秒读完成次数
				parti.WioPerSecond = float64(wio-parti.Wio) / difftime                     //平均每秒写完成次数
				parti.RsectPerSecond = float64(rsect-parti.Rsect) / difftime               //平均每秒读扇区次数
				parti.WsectPerSecond = float64(wsect-parti.Wsect) / difftime               //平均每秒写扇区次数
				parti.RkbPerSecond = parti.RsectPerSecond / 2                              //每秒读kb数(一扇区为512bytes)
				parti.WkbPerSecond = parti.WsectPerSecond / 2                              //每秒写kb数
				parti.QueueSz = float64(aveq-parti.Aveq) / difftime / 1000.0               //平均I/O队列长度
				parti.ReqRate = float64(elapsed-parti.Elapsed) * 100.0 / difftime / 1000.0 //平均I/O操作百分比

				RkbSum += parti.RkbPerSecond
				WkbSum += parti.WkbPerSecond
				QueueSzSum += parti.QueueSz
				RmergeSum += parti.RmergePerSecond
				WmergeSum += parti.WmergePerSecond
				RioSum += parti.RioPerSecond
				WioSum += parti.WioPerSecond
				RsectSum += parti.RsectPerSecond
				WsectSum += parti.WsectPerSecond
				ReqRateSum += parti.ReqRate

				if parti.ReqRate >= maxReqRate { //>=,都为0时做初始化
					maxReqRate = parti.ReqRate
					maxReqRateParti = partiName
					//maxReqRateName = partiName
				}

			} else {
				parti.RmergePerSecond = 0
				parti.WmergePerSecond = 0
				parti.RioPerSecond = 0
				parti.WioPerSecond = 0
				parti.RsectPerSecond = 0
				parti.RkbPerSecond = 0
				parti.WsectPerSecond = 0
				parti.QueueSz = 0
				parti.ReqRate = 0

				if parti.ReqRate >= maxReqRate {
					maxReqRate = parti.ReqRate
					maxReqRateParti = partiName
					//maxReqRateName = partiName
				}
			}
			diffio := float64(rio - parti.Rio + wio - parti.Wio)
			if diffio > 0 {
				parti.ReqSz = float64(rsect+wsect-parti.Rsect-parti.Wsect) / diffio                    //平均I/O大小
				parti.AwaitElapsed = float64(relapsed+welapsed-parti.Relapsed-parti.Welapsed) / diffio //平均I/O等待时间
				parti.ServeElapsed = float64(elapsed-parti.Elapsed) / diffio                           //平均I/O服务时间
				ReqSzSum += parti.ReqSz
				ServeSum += parti.ServeElapsed
				AwaitSum += parti.AwaitElapsed
			} else {
				parti.ReqSz = 0
				parti.AwaitElapsed = 0
				parti.ServeElapsed = 0
			}
		}

		parti.Last = now //更新采集时间
		parti.Rio = rio
		parti.Rmerge = rmerge
		parti.Rsect = rsect
		parti.Relapsed = relapsed
		parti.Wio = wio
		parti.Wmerge = wmerge
		parti.Wsect = wsect
		parti.Welapsed = welapsed
		parti.Aveq = aveq
		parti.Elapsed = elapsed
	}

	partiLen := float64(len(this.PartiMap))
	if partiLen > 0 {
		this.QueueSzAvg = QueueSzSum / partiLen //所有分区平均I/O队列长度
		this.ReqSzAvg = ReqSzSum / partiLen     //所有分区平均I/O大小
		this.ServeAvg = ServeSum / partiLen     //所有分区平均I/O服务时间
		this.AwaitAvg = AwaitSum / partiLen     //所有分区平均I/O等待时间
		this.RkbAvg = RkbSum / partiLen         //所有分区平均每秒读kb数
		this.WkbAvg = WkbSum / partiLen         //所有分区平均每秒写kb数
		this.RmergeAvg = RmergeSum / partiLen   //所有分区平均每秒merge读次数
		this.WmergeAvg = WmergeSum / partiLen   //所有分区平均每秒merge写次数
		this.RioAvg = RioSum / partiLen         //所有分区平均每秒读完成次数
		this.WioAvg = WioSum / partiLen         //所有分区平均每秒写完成次数
		this.RsectAvg = RsectSum / partiLen     //所有分区平均每秒读扇区次数
		this.WsectAvg = WsectSum / partiLen     //所有分区平均每秒写扇区次数
		this.ReqRateAvg = ReqRateSum / partiLen //所有分区平均I/O操作百分比
	}

	this.MaxReqRate = maxReqRate
	this.MaxReqRateParti = maxReqRateParti
	//this.MaxReqRateName = maxReqRateName

	return nil
}

//磁盘各个分区平均I/O队列长度
func (this *DiskIO) QueueSzSetFunc(args string) string {
	queueSzSet := []string{}
	for _, parti := range this.PartiMap {
		queueSzSet = append(queueSzSet, parti.Name+"|"+strconv.FormatFloat(parti.QueueSz, 'f', 2, 64))
	}
	return strings.Join(queueSzSet, "$") + "$"
}

//磁盘所有分区平均I/O队列长度
func (this *DiskIO) QueueSzAvgFunc(args string) string {
	return FloatToString(this.QueueSzAvg)
}

//磁盘各个分区平均I/O大小
func (this *DiskIO) ReqSzSetFunc(args string) string {
	reqSzSet := []string{}
	for _, parti := range this.PartiMap {
		reqSzSet = append(reqSzSet, parti.Name+"|"+strconv.FormatFloat(parti.ReqSz, 'f', 2, 64))
	}
	return strings.Join(reqSzSet, "$") + "$"
}

//磁盘所有分区平均I/O大小
func (this *DiskIO) ReqSzAvgFunc(args string) string {
	return FloatToString(this.ReqSzAvg)
}

//磁盘各个分区平均I/O服务时间(ms)
func (this *DiskIO) ServeSetFunc(args string) string {
	serveSet := []string{}
	for _, parti := range this.PartiMap {
		serveSet = append(serveSet, parti.Name+"|"+strconv.FormatFloat(parti.ServeElapsed, 'f', 2, 64))
	}
	return strings.Join(serveSet, "$") + "$"
}

//磁盘所有分区平均I/O服务时间(ms)
func (this *DiskIO) ServeAvgFunc(args string) string {
	return FloatToString(this.ServeAvg)
}

//磁盘各个分区平均I/O等待时间
func (this *DiskIO) AwaitSetFunc(args string) string {
	awaitSet := []string{}
	for _, parti := range this.PartiMap {
		awaitSet = append(awaitSet, parti.Name+"|"+strconv.FormatFloat(parti.AwaitElapsed, 'f', 2, 64))
	}
	return strings.Join(awaitSet, "$") + "$"
}

//磁盘所有分区平均I/O等待时间
func (this *DiskIO) AwaitAvgFunc(args string) string {
	return FloatToString(this.AwaitAvg)
}

//磁盘各个分区每秒读kb数(kb)
func (this *DiskIO) RkbPerSecondSetFunc(args string) string {
	rkbSet := []string{}
	for _, parti := range this.PartiMap {
		rkbSet = append(rkbSet, parti.Name+"|"+strconv.FormatFloat(parti.RkbPerSecond, 'f', 2, 64))
	}
	return strings.Join(rkbSet, "$") + "$"
}

//磁盘所有分区平均每秒读kb数(kb)
func (this *DiskIO) RkbPerSecondFunc(args string) string {
	return FloatToString(this.RkbAvg)
}

//磁盘各个分区每秒merge读次数
func (this *DiskIO) RmergePerSecondSetFunc(args string) string {
	rmergeSet := []string{}
	for _, parti := range this.PartiMap {
		rmergeSet = append(rmergeSet, parti.Name+"|"+strconv.FormatFloat(parti.RmergePerSecond, 'f', 2, 64))
	}
	return strings.Join(rmergeSet, "$") + "$"
}

//磁盘所有分区平均每秒merge读次数
func (this *DiskIO) RmergePerSecondFunc(args string) string {
	return FloatToString(this.RmergeAvg)
}

//磁盘各个分区每秒读I/O次数
func (this *DiskIO) RioPerSecondSetFunc(args string) string {
	rioSet := []string{}
	for _, parti := range this.PartiMap {
		rioSet = append(rioSet, parti.Name+"|"+strconv.FormatFloat(parti.RioPerSecond, 'f', 2, 64))
	}
	return strings.Join(rioSet, "$") + "$"
}

//磁盘所有分区平均每秒读次数
func (this *DiskIO) RioPerSecondFunc(args string) string {
	return FloatToString(this.RioAvg)
}

//磁盘各个分区平均每秒读扇区数
func (this *DiskIO) RsectPerSecondSetFunc(args string) string {
	rsectSet := []string{}
	for _, parti := range this.PartiMap {
		rsectSet = append(rsectSet, parti.Name+"|"+strconv.FormatFloat(parti.RsectPerSecond, 'f', 2, 64))
	}
	return strings.Join(rsectSet, "$") + "$"
}

//磁盘所有分区平均每秒读扇区数
func (this *DiskIO) RsectPerSecondFunc(args string) string {
	return FloatToString(this.RsectAvg)
}

//磁盘所有分区平均I/O操作百分比
func (this *DiskIO) ReqRateAvgFunc(args string) string {
	return FloatToString(this.ReqRateAvg)
}

//磁盘各个分区每秒写kb数(kb)
func (this *DiskIO) WkbPerSecondSetFunc(args string) string {
	wkbSet := []string{}
	for _, parti := range this.PartiMap {
		wkbSet = append(wkbSet, parti.Name+"|"+strconv.FormatFloat(parti.WkbPerSecond, 'f', 2, 64))
	}
	return strings.Join(wkbSet, "$") + "$"
}

//磁盘所有分区平均每秒写kb数(kb)
func (this *DiskIO) WkbPerSecondFunc(args string) string {
	return FloatToString(this.WkbAvg)
}

//磁盘各个分区每秒merge写次数
func (this *DiskIO) WmergePerSecondSetFunc(args string) string {
	wmergeSet := []string{}
	for _, parti := range this.PartiMap {
		wmergeSet = append(wmergeSet, parti.Name+"|"+strconv.FormatFloat(parti.WmergePerSecond, 'f', 2, 64))
	}
	return strings.Join(wmergeSet, "$") + "$"
}

//磁盘所有分区平均每秒merge写次数
func (this *DiskIO) WmergePerSecondFunc(args string) string {
	return FloatToString(this.WmergeAvg)
}

//磁盘各个分区每秒写次数
func (this *DiskIO) WioPerSecondSetFunc(args string) string {
	wioSet := []string{}
	for _, parti := range this.PartiMap {
		wioSet = append(wioSet, parti.Name+"|"+strconv.FormatFloat(parti.WioPerSecond, 'f', 2, 64))
	}
	return strings.Join(wioSet, "$") + "$"
}

//磁盘所有分区平均每秒写次数
func (this *DiskIO) WioPerSecondFunc(args string) string {
	return FloatToString(this.WioAvg)
}

//磁盘各个分区平均每秒写扇区数
func (this *DiskIO) WsectPerSecondSetFunc(args string) string {
	wsectSet := []string{}
	for _, parti := range this.PartiMap {
		wsectSet = append(wsectSet, parti.Name+"|"+strconv.FormatFloat(parti.WsectPerSecond, 'f', 2, 64))
	}
	return strings.Join(wsectSet, "$") + "$"
}

//磁盘所有分区平均每秒写扇区数
func (this *DiskIO) WsectPerSecondFunc(args string) string {
	return FloatToString(this.WsectAvg)
}

func (this *DiskIO) GetKeyByIndex(args string) (string, error) {
	index, err := strconv.Atoi(args)
	if err != nil {
		return "", err
	}
	if index < 0 {
		return "", errors.New("invalid index")
	}
	length := len(this.PartiNames)
	if index > length-1 {
		return "", errors.New("invalid index")
	}
	return this.PartiNames[index], nil
}

//磁盘n平均I/O队列长度
func (this *DiskIO) DiskQueueSzAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.QueueSz)
}

//磁盘n平均I/O大小
func (this *DiskIO) DiskReqSzAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.ReqSz)
}

//磁盘n平均I/O服务时间(ms)
func (this *DiskIO) DiskServeAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.ServeElapsed)
}

//磁盘n平均I/O等待时间
func (this *DiskIO) DiskAwaitAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.AwaitElapsed)
}

//磁盘n每秒读kb数
func (this *DiskIO) DiskRkbAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.RkbPerSecond)
}

//磁盘n每秒merge读次数
func (this *DiskIO) DiskRmergeAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.RmergePerSecond)
}

//磁盘n每秒读完成次数
func (this *DiskIO) DiskRioAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.RioPerSecond)
}

//磁盘n每秒读扇区数
func (this *DiskIO) DiskRsectAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.RsectPerSecond)
}

//磁盘n每秒I/O操作百分比
func (this *DiskIO) DiskReqRateAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.ReqRate)
}

//磁盘n每秒写kb数
func (this *DiskIO) DiskWkbAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.WkbPerSecond)
}

//磁盘n每秒merge写次数
func (this *DiskIO) DiskWmergeAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.WmergePerSecond)
}

//磁盘n每秒写完成次数
func (this *DiskIO) DiskWioAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.WioPerSecond)
}

//磁盘n每秒写扇区数
func (this *DiskIO) DiskWsectAvgFunc(args string) string {
	key, err := this.GetKeyByIndex(args)
	if err != nil {
		return ""
	}
	parti, exists := this.PartiMap[key]
	if !exists {
		return ""
	}
	return FloatToString(parti.WsectPerSecond)
}

//磁盘各个分区I/O操作百分比
func (this *DiskIO) ReqRateSetFunc(args string) string {
	reqRateSet := []string{}
	for _, parti := range this.PartiMap {
		reqRateSet = append(reqRateSet, parti.Name+"|"+FloatToString(parti.ReqRate))
	}
	return strings.Join(reqRateSet, "$") + "$"
}

//磁盘I/O最大使用率
func (this *DiskIO) MaxUsedRateFunc(args string) string {
	return FloatToString(this.MaxReqRate) + "," + this.MaxReqRateParti
}
