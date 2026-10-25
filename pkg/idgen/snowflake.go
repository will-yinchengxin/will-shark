// Package idgen 提供 UUID 与雪花算法 ID 生成。
package idgen

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Twitter 的 Snowflake 算法 的实现
/*
SnowFlake 的结构如下（每部分用-分开）:
0 - 0000000000 0000000000 0000000000 0000000000 0 - 00000 - 00000 - 000000000000
- 最高位是符号位，正数是 0，负数是 1， id 一般是正数，因此最高位固定是 0；
- 41 位时间戳（毫秒级），注意，41 位时间戳不是存储当前时间的时间戳，而是存储时间戳的差值（当前时间戳 - 开始时间戳），这样能存的时间更长，开始时间一般指定为项目启动时间，由程序指定。可以使用 69 年：`(1<<41)/(1000*60*60*24*365)`；
- 10 位的机器相关位，可以部署在 1024 个节点，包括 5 位的 datacenterId（数据中心 ID） 和 5 位 workerId（工作机器 ID）；
- 12 位系列号，毫秒内的计数。12 位的计数顺序号支持每个节点每毫秒（同一机器，同一时间戳）产生 4096 个 ID 序号；
*/
const (
	sequenceMask   = 1<<12 - 1
	workerMask     = 1<<5 - 1
	dataCenterMask = 1<<5 - 1

	workerLeftShift     = 12
	dataCenterLeftShift = 17
	timestampLeftShift  = 22
)

type SnowFlake struct {
	mutex sync.Mutex

	sequence     int16
	dataCenterID uint8
	workerID     uint8

	// 上次生成 ID 的时间戳（毫秒）
	lastTimestamp int64

	startTime time.Time
}

// defaultEpoch New 使用的固定起始时间。起始时间必须固定不变，否则重启后生成的 ID 会变小并与历史 ID 重复
var defaultEpoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// NewWith 给定开始时间和可选的 dataCenterID 和 workerID（注意两者的顺序）
// 如果 ids 没传，则使用 machineID。startTime 一经使用不能修改。
func NewWith(startTime time.Time, ids ...uint8) *SnowFlake {
	var dataCenterID, workerID uint8

	idLen := len(ids)
	if idLen >= 2 {
		dataCenterID, workerID = ids[0], ids[1]
	} else if idLen == 1 {
		dataCenterID = ids[0]
		workerID = ids[0]
	} else {
		dataCenterID, workerID = machineID()
	}

	return &SnowFlake{
		startTime:    startTime.UTC(),
		dataCenterID: dataCenterID & dataCenterMask,
		workerID:     workerID & workerMask,
	}
}

// New 使用固定起始时间（2024-01-01 UTC）与本机 IP 推导的机器号
func New() *SnowFlake {
	dataCenterID, workerID := machineID()
	return NewWith(defaultEpoch, dataCenterID, workerID)
}

// NextID 获取一个 ID；并发安全，同一实例生成的 ID 严格递增
func (s *SnowFlake) NextID() int64 {
	s.mutex.Lock()

	millisecond := genMillisecond()
	// 时钟回拨：沿用上次的时间戳继续递增序号，保证不重复（不 panic）
	if millisecond < s.lastTimestamp {
		millisecond = s.lastTimestamp
	}
	// 同一毫秒，进行毫秒内序号递增
	if millisecond == s.lastTimestamp {
		s.sequence = (s.sequence + 1) & sequenceMask
		// 当前毫秒内序号用完，借用下一毫秒
		if s.sequence == 0 {
			millisecond++
		}
	} else {
		// 时间戳改变，毫秒内序号重置
		s.sequence = 0
	}
	s.lastTimestamp = millisecond
	sequence := s.sequence

	s.mutex.Unlock()

	elaspedMillisecond := millisecond - s.startTime.UnixNano()/1e6

	return elaspedMillisecond<<timestampLeftShift |
		int64(s.dataCenterID)<<dataCenterLeftShift |
		int64(s.workerID)<<workerLeftShift |
		int64(sequence)
}

func (s *SnowFlake) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return fmt.Sprintf("start_time:%s, data_center:%d, worker_id:%d, sequence:%d",
		s.startTime, s.dataCenterID, s.workerID, s.sequence)
}

func machineID() (uint8, uint8) {
	as, err := net.InterfaceAddrs()
	if err != nil {
		return 0, 0
	}

	for _, a := range as {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}

		// 取 IPv4 的低 10 位：同一 /22 网段内各机器的机器号不同
		if ip := ipnet.IP.To4(); ip != nil {
			low := uint16(ip[2])<<8 | uint16(ip[3])
			return uint8((low >> 5) & dataCenterMask), uint8(low & workerMask)
		}
	}

	return 0, 0
}

// genMillisecond 获取当前 UTC 时间的时间戳（毫秒表示）
func genMillisecond() int64 {
	return time.Now().UTC().UnixNano() / 1e6
}
