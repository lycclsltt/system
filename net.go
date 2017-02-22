package system

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Ifi struct {
	Name              string
	Ip                string
	Speed             float64
	OutRecvPkgErrRate float64 //外网收包错误率
	OutSendPkgErrRate float64 //外网发包错误率
	RecvByte          uint64  //接收的字节数
	RecvPkg           uint64  //接收正确的包数
	RecvErr           uint64  //接收错误的包数
	SendByte          uint64  //发送的字节数
	SendPkg           uint64  //发送正确的包数
	SendErr           uint64  //发送错误的包数

	RecvByteAvg float64 //一个周期平均每秒接收字节数
	SendByteAvg float64 //一个周期平均每秒发送字节数
	RecvErrRate float64 //一个周期收包错误率
	SendErrRate float64 //一个周期发包错误率
	RecvPkgAvg  float64 //一个周期平均每秒收包数
	SendPkgAvg  float64 //一个周期平均每秒发包数

	Last int64 //上次采集时间
}

//判断是否为内网
func (this *Ifi) IsInEth() bool {
	if this.Ip == "" {
		return false
	}
	fields := strings.Split(this.Ip, ".")
	if len(fields) != 4 {
		return false
	}
	field1 := fields[0]
	if strings.Contains(field1, "10") ||
		strings.Contains(field1, "127") ||
		strings.Contains(field1, "192") ||
		strings.Contains(field1, "172") {
		return true
	}
	return false
}

type NetWork struct {
	IfiMap      map[string]*Ifi
	IfiNames    []string
	RecvByteSum float64 //所有内外网网络接口一个周期平均接收字节数之和
	SendByteSum float64 //所有内外网网络接口一个周期平均发送字节数之和

	InRecvByteSum    float64 //所有内网网络接口平均每秒接收字节数之和
	InSendByteSum    float64 //所有内网网络接口平均每秒发送字节数之和
	InRecvPkgSum     float64 //所有内网网络接口平均每秒收包数之和
	InSendPkgSum     float64 //所有内网网络接口平均每秒发包数之和
	InRecvErrRateSum float64 //所有内网网络接口收包错误率之和
	InSendErrRateSum float64 //所有内网网络接口发包错误率之和

	OutRecvByteSum    float64 //所有外网网络接口平均每秒接收字节数之和
	OutSendByteSum    float64 //所有外网网络接口平均每秒发送字节数之和
	OutRecvPkgSum     float64 //所有外网网络接口平均每秒收包数之和
	OutSendPkgSum     float64 //所有外网网络接口平均每秒发包数之和
	OutRecvErrRateSum float64 //所有外网网络接口收包错误率之和
	OutSendErrRateSum float64 //所有外网网络接口发包错误率之和

	EthInMaxUseRate  float64 //内网网卡使用率
	EthOutMaxUseRate float64 //外网网卡使用率

	RecvSendDetail string //收发接口收发字节数详细信息
	ModelDetail    string //网络接口型号带宽详细信息

	/*
		//外网网卡流入环比
		OutRecvByteSum10Sum   float64 //外网网卡平均每秒接收字节累加和
		OutRecvByteSum10Times int     //外网网卡平均每秒接收字节累加次数
		OutRecvByteSum10      float64 //外网网卡流入10分钟环比
		OutRecvByteSum10Last  int64

		OutRecvByteSum60Sum   float64 //外网网卡平均每秒接收字节累加和
		OutRecvByteSum60Times int     //外网网卡平均每秒接收字节累加次数
		OutRecvByteSum60      float64 //外网网卡流入60分钟环比
		OutRecvByteSum60Last  int64

		OutRecvByteSumDaySum   float64 //外网网卡平均每秒接收字节累加和
		OutRecvByteSumDayTimes int     //外网网卡平均每秒接收字节累加次数
		OutRecvByteSumDay      float64 //外网网卡流入日同比
		OutRecvByteSumDayLast  int64
	*/
}

func (this *NetWork) Collect() error {
	this.ResetIfiData()
	return this.InitNetWorkInfo()
}

func (this *NetWork) ResetIfiData() {
	this.RecvByteSum = 0
	this.SendByteSum = 0

	this.InRecvByteSum = 0
	this.InSendByteSum = 0
	this.InRecvPkgSum = 0
	this.InSendPkgSum = 0
	this.InRecvErrRateSum = 0
	this.InSendErrRateSum = 0

	this.OutRecvByteSum = 0
	this.OutSendByteSum = 0
	this.OutRecvPkgSum = 0
	this.OutSendPkgSum = 0
	this.OutRecvErrRateSum = 0
	this.OutSendErrRateSum = 0

	this.EthInMaxUseRate = 0
	this.EthOutMaxUseRate = 0

	this.RecvSendDetail = ""
	this.ModelDetail = ""
}

func (this *NetWork) InitNetWorkInfo() error {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if !strings.Contains(line, ":") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}

		ethname := fields[0]
		Trim(&ethname)

		fields = strings.Fields(fields[1])
		if len(fields) != 16 {
			continue
		}
		recvByte, _ := strconv.ParseUint(fields[0], 10, 64)
		recvPkg, _ := strconv.ParseUint(fields[1], 10, 64)
		recvErr, _ := strconv.ParseUint(fields[2], 10, 64)
		sendByte, _ := strconv.ParseUint(fields[8], 10, 64)
		sendPkg, _ := strconv.ParseUint(fields[9], 10, 64)
		sendErr, _ := strconv.ParseUint(fields[10], 10, 64)

		//根据网卡名得到对应的网络接口
		netifi, err := net.InterfaceByName(ethname)
		if err != nil {
			continue
		}
		var addrs []net.Addr
		addrs, err = netifi.Addrs()
		if err != nil {
			continue
		}
		if len(addrs) == 0 {
			continue
		}
		moniTag := true
		for _, addr := range addrs {
			cidr := addr.String()
			if strings.Contains(cidr, "0.0.0.0") || strings.Contains(cidr, "127.0.0.1") {
				//0.0.0.0 127.0.0.1 不监控
				moniTag = false
				break
			}
		}
		if moniTag == false {
			continue
		}
		_, exists := this.IfiMap[ethname]
		if !exists {
			this.IfiMap[ethname] = &Ifi{}
			this.IfiNames = append(this.IfiNames, ethname)
		}
		ifi, _ := this.IfiMap[ethname]

		var (
			recvByteAvg float64
			recvPkgAvg  float64
			recvErrRate float64
			sendByteAvg float64
			sendPkgAvg  float64
			sendErrRate float64
		)
		now := time.Now().Unix()
		difftime := float64(now - ifi.Last)
		if ifi.Last == 0 {
			//第一次采集，没有时间差，不计算
		} else {
			if difftime > 0 {
				recvByteAvg = float64(recvByte-ifi.RecvByte) / difftime //平均每秒接收字节数
				recvPkgAvg = float64(recvPkg-ifi.RecvPkg) / difftime    //平均每秒接收正确的包数
				if recvPkg-ifi.RecvPkg > 0 {
					recvErrRate = float64(recvErr-ifi.RecvErr) / float64(recvPkg-ifi.RecvPkg) //一个周期收包错误率
				}
				sendByteAvg = float64(sendByte-ifi.SendByte) / difftime //平均每秒发送字节数
				sendPkgAvg = float64(sendPkg-ifi.SendPkg) / difftime    //平均每秒发送正确的包数
				if sendPkg-ifi.SendPkg > 0 {
					sendErrRate = float64(sendErr-ifi.SendErr) / float64(sendPkg-ifi.SendPkg) //一个周期发包错误率
				}
			}
		}

		ifi.Name = ethname
		ifi.Ip = addrs[0].String()
		ifi.RecvByte = recvByte
		ifi.RecvPkg = recvPkg
		ifi.RecvErr = recvErr
		ifi.SendByte = sendByte
		ifi.SendPkg = sendPkg
		ifi.SendErr = sendErr
		ifi.RecvByteAvg = recvByteAvg
		ifi.SendByteAvg = sendByteAvg
		ifi.RecvErrRate = recvErrRate
		ifi.SendErrRate = sendErrRate
		ifi.RecvPkgAvg = recvPkgAvg
		ifi.SendPkgAvg = sendPkgAvg
		ifi.Last = now

		if ifi.IsInEth() {
			//内网
			this.InRecvByteSum += recvByteAvg
			this.InSendByteSum += sendByteAvg
			this.InRecvPkgSum += recvPkgAvg
			this.InSendPkgSum += sendPkgAvg
			this.InRecvErrRateSum += recvErrRate
			this.InSendErrRateSum += sendErrRate
		} else {
			//外网
			this.OutRecvByteSum += recvByteAvg
			this.OutSendByteSum += sendByteAvg
			this.OutRecvPkgSum += recvPkgAvg
			this.OutSendPkgSum += sendPkgAvg
			this.OutRecvErrRateSum += recvErrRate
			this.InSendErrRateSum += sendErrRate
		}

		this.RecvByteSum += recvByteAvg
		this.SendByteSum += sendByteAvg
		this.RecvSendDetail += ifi.Ip + "=" + ifi.Name + "=(" + strconv.FormatFloat(recvByteAvg, 'f', 0, 64) + "|" +
			strconv.FormatFloat(sendByteAvg, 'f', 0, 64) + ")$"

		cmd := fmt.Sprintf("/sbin/ethtool %s 2>/dev/null", ethname)
		output, err := Exec(cmd)
		if err != nil {
			continue
		}
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Speed") {
				fields := strings.Split(line, ":")
				if len(fields) != 2 {
					continue
				}
				field2 := fields[1]
				Trim(&field2)
				field2 = strings.Replace(field2, "Mb/s", "", -1)
				speed, err := strconv.ParseFloat(field2, 64) //Mb/s, 注意是小b
				if err != nil {
					continue
				}
				ifi.Speed = speed
				if speed > 0 {
					inEthUseRate := float64(recvByteAvg*8*100) / float64(speed*1024*1024)
					if inEthUseRate > this.EthInMaxUseRate {
						this.EthInMaxUseRate = inEthUseRate
					}
					outEthUseRate := float64(sendByteAvg*8*100) / float64(speed*1024*1024)
					if outEthUseRate > this.EthOutMaxUseRate {
						this.EthOutMaxUseRate = outEthUseRate
					}
				}
				break
			}
		}
		this.ModelDetail += ifi.Name + "|" + ifi.Ip + "|" + FloatToString(ifi.Speed) + "$"
	}
	return nil
}

//外网收包错误率
func (this *NetWork) OutRecvErrRateSumFunc(args string) string {
	return FloatToString(this.OutRecvErrRateSum)
}

//外网发包错误率
func (this *NetWork) OutSendErrRateSumFunc(args string) string {
	return FloatToString(this.OutSendErrRateSum)
}

//外网发包速度(pkg/s)
func (this *NetWork) OutSendPkgSumFunc(args string) string {
	return FloatToString(this.OutSendPkgSum)
}

//外网收包速度(pkg/s)
func (this *NetWork) OutRecvPkgSumFunc(args string) string {
	return FloatToString(this.OutRecvPkgSum)
}

//内网收包错误率
func (this *NetWork) InRecvErrRateSumFunc(args string) string {
	return FloatToString(this.InRecvErrRateSum)
}

//内网发包错误率
func (this *NetWork) InSendErrRateSumFunc(args string) string {
	return FloatToString(this.InSendErrRateSum)
}

//内网发包速度(pkg/s)
func (this *NetWork) InSendPkgSumFunc(args string) string {
	return FloatToString(this.InSendPkgSum)
}

//内网收包速度
func (this *NetWork) InRecvPkgSumFunc(args string) string {
	return FloatToString(this.InRecvPkgSum)
}

//网卡入带宽最大使用率
func (this *NetWork) EthInMaxUseRateFunc(args string) string {
	return FloatToString(this.EthInMaxUseRate)
}

//网卡出带宽最大使用率
func (this *NetWork) EthOutMaxUseRateFunc(args string) string {
	return FloatToString(this.EthOutMaxUseRate)
}

func (this *NetWork) GetIfiByIndex(args string) (*Ifi, error) {
	index, err := strconv.Atoi(args)
	if err != nil {
		return nil, err
	}
	if index < 0 {
		return nil, errors.New("invalid index")
	}
	length := len(this.IfiNames)
	if index > length-1 {
		return nil, errors.New("invalid index")
	}
	key := this.IfiNames[index]
	ifi, exists := this.IfiMap[key]
	if exists {
		return ifi, nil
	}
	return nil, errors.New("key not found")
}

//接收速率(byte/s)
func (this *NetWork) EthRecvByteAvgFunc(args string) string {
	ifi, err := this.GetIfiByIndex(args)
	if err != nil {
		return ""
	}
	return strconv.FormatFloat(ifi.RecvByteAvg, 'f', 0, 64)
}

//包接收速率(pkg/s)
func (this *NetWork) EthRecvPkgAvgFunc(args string) string {
	ifi, err := this.GetIfiByIndex(args)
	if err != nil {
		return ""
	}
	return strconv.FormatFloat(ifi.RecvPkgAvg, 'f', 0, 64)
}

//发送速率(byte/s)
func (this *NetWork) EthSendByteAvgFunc(args string) string {
	ifi, err := this.GetIfiByIndex(args)
	if err != nil {
		return ""
	}
	return strconv.FormatFloat(ifi.SendByteAvg, 'f', 0, 64)
}

//包发送速率(pkg/s)
func (this *NetWork) EthSendPkgAvgFunc(args string) string {
	ifi, err := this.GetIfiByIndex(args)
	if err != nil {
		return ""
	}
	return strconv.FormatFloat(ifi.SendPkgAvg, 'f', 0, 64)
}

//收包错误率
func (this *NetWork) EthRecvErrRateFunc(args string) string {
	ifi, err := this.GetIfiByIndex(args)
	if err != nil {
		return ""
	}
	return FloatToString(ifi.RecvErrRate)
}

//网卡速率(Mb/s)
func (this *NetWork) EthSpeedFunc(args string) string {
	ifi, err := this.GetIfiByIndex(args)
	if err != nil {
		return ""
	}
	return strconv.FormatFloat(ifi.Speed, 'f', 0, 64)
}

//发包错误率
func (this *NetWork) EthSendErrRateFunc(args string) string {
	ifi, err := this.GetIfiByIndex(args)
	if err != nil {
		return ""
	}
	return FloatToString(ifi.SendErrRate)
}

//EthModelFunc ... 机器网卡信息
func (this *NetWork) EthModelFunc(args string) string {
	return this.ModelDetail
}

/*
func (this *NetWork) AddRecvBytes(bytes float64) {
	this.OutRecvByteSum10Sum += bytes
	this.OutRecvByteSum10Times++
	this.OutRecvByteSum60Sum += bytes
	this.OutRecvByteSum60Times++
	this.OutRecvByteSumDaySum += bytes
	this.OutRecvByteSumDayTimes++
}*/

/*
func (this *NetWork) ResetRecvSum10() {
	this.OutRecvByteSum10Sum = 0
	this.OutRecvByteSum10Times = 0
}

func (this *NetWork) ResetRecvSum60() {
	this.OutRecvByteSum60Sum = 0
	this.OutRecvByteSum60Times = 0
}

func (this *NetWork) ResetRecvSumDay() {
	this.OutRecvByteSumDaySum = 0
	this.OutRecvByteSumDayTimes = 0
}*/

//外网网卡流入流量(byte/s)
func (this *NetWork) OutEthRecvByteAvgFunc(args string) string {
	//this.AddRecvBytes(this.OutRecvByteSum)
	return strconv.FormatFloat(this.OutRecvByteSum, 'f', 0, 64)
}

//外网网卡流出流量(byte/s)
func (this *NetWork) OutEthSendByteAvgFunc(args string) string {
	return strconv.FormatFloat(this.OutSendByteSum, 'f', 0, 64)
}

//内网网卡流入流量(byte/s)
func (this *NetWork) InEthRecvByteAvgFunc(args string) string {
	return strconv.FormatFloat(this.InRecvByteSum, 'f', 0, 64)
}

//内网网卡流出流量(byte/s)
func (this *NetWork) InEthSendByteAvgFunc(args string) string {
	return strconv.FormatFloat(this.InSendByteSum, 'f', 0, 64)
}

//所有网卡流入流量(byte/s)
func (this *NetWork) AllEthRecvByteAvgFunc(args string) string {
	return strconv.FormatFloat(this.RecvByteSum, 'f', 0, 64)

}

//所有网卡流出流量(byte/s)
func (this *NetWork) EthRecvSendAvgFunc(args string) string {
	return strconv.FormatFloat(this.SendByteSum, 'f', 0, 64)
}

//所有网卡流量信息
func (this *NetWork) EthByteSetFunc(args string) string {
	return this.RecvSendDetail
}

/*
//外网网卡流入，10分钟环比
func (this *NetWork) OutEthRecv10Func(args string) string {
	if time.Now().Unix()-this.OutRecvByteSum10Last < 600 {
		return ""
	}
	if this.OutRecvByteSum10Times <= 0 {
		return ""
	}
	var ret float64 = 0
	avg := this.OutRecvByteSum10Sum / float64(this.OutRecvByteSum10Times)
	//到这
	if this.OutRecvByteSum10 == 0 && avg != 0 {
		ret = 100
	} else {
		ret = (avg - this.OutRecvByteSum10) / this.OutRecvByteSum10 * 100
	}
	this.OutRecvByteSum10 = avg
	this.OutRecvByteSum10Last = time.Now().Unix()
	this.ResetRecvSum10()
	return g.FloatToString(ret)
}

//外网网卡流入，60分钟环比
func (this *NetWork) OutEthRecv60Func(args string) string {
	if time.Now().Unix()-this.OutRecvByteSum60Last < 3600 {
		return ""
	}
	if this.OutRecvByteSum60Times <= 0 {
		return ""
	}
	var ret float64 = 0
	avg := this.OutRecvByteSum60Sum / float64(this.OutRecvByteSum60Times)
	//到这
	if this.OutRecvByteSum60 == 0 && avg != 0 {
		ret = 100
	} else {
		ret = (avg - this.OutRecvByteSum60) / this.OutRecvByteSum60 * 100
	}
	this.OutRecvByteSum60 = avg
	this.OutRecvByteSum60Last = time.Now().Unix()
	this.ResetRecvSum60()
	return g.FloatToString(ret)
}

//外网网卡流入，日环比
func (this *NetWork) OutEthRecvDayFunc(args string) string {
	if time.Now().Unix()-this.OutRecvByteSumDayLast < 3600 {
		return ""
	}
	if this.OutRecvByteSumDayTimes <= 0 {
		return ""
	}
	var ret float64 = 0
	avg := this.OutRecvByteSumDaySum / float64(this.OutRecvByteSumDayTimes)
	//到这
	if this.OutRecvByteSumDay == 0 && avg != 0 {
		ret = 100
	} else {
		ret = (avg - this.OutRecvByteSumDay) / this.OutRecvByteSumDay * 100
	}
	this.OutRecvByteSumDay = avg
	this.OutRecvByteSumDayLast = time.Now().Unix()
	this.ResetRecvSumDay()
	return g.FloatToString(ret)
}
*/

func ConnNumByPort(port string) string {
	return ExecOutput("netstat -pnt |grep ':" + port + "\\b' |wc -l")
}
