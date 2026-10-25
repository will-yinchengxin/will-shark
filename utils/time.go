package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	GoBronTime        = "2006-01-02 15:04:05"
	GoBronTimeOnlyDay = "2006-01-02"
)

func NowData() time.Time {
	return time.Now()
}

func Now() int64 {
	return time.Now().Unix()
}

// ParseTimeStrToInt 解析 "2006-01-02 15:04:05"（本地时区）为秒级时间戳；格式错误返回 0
func ParseTimeStrToInt(timeStr string) int64 {
	parseTime, err := time.ParseInLocation(GoBronTime, timeStr, time.Local)
	if err != nil {
		return 0
	}
	return parseTime.Unix()
}

func ParseTimeIntToStr(timeInt int64) string {
	return time.Unix(timeInt, 0).Local().Format(GoBronTime)
}

func ParseTimeIntToStrOnlyDay(timeInt int64) string {
	return time.Unix(timeInt, 0).Local().Format(GoBronTimeOnlyDay)
}

func GetTimeStrSlice(beginDate, endDate string) []string {
	format := "2006-01-02"
	bDate, err1 := time.ParseInLocation(format, beginDate, time.Local)
	eDate, err2 := time.ParseInLocation(format, endDate, time.Local)
	if err1 != nil || err2 != nil {
		return []string{}
	}
	// 闭区间 [beginDate, endDate]；按日历日递增，不受夏令时影响
	dlist := make([]string, 0)
	for d := bDate; !d.After(eDate); d = d.AddDate(0, 0, 1) {
		dlist = append(dlist, d.Format(format))
	}
	return dlist
}

// AgoUnixMilli 返回"现在往前推 duration"的毫秒时间戳，duration 为数字 + 单位：
// h 小时、d 天、w 周、m 月（按 30 天）、y 年，如 "7d"。注意 m 是月而不是分钟，与 time.ParseDuration 不同。
func AgoUnixMilli(duration string) (int64, error) {
	duration = strings.ToLower(duration)
	if len(duration) < 2 {
		return 0, fmt.Errorf("invalid duration format: %s", duration)
	}

	value := duration[:len(duration)-1]
	unit := duration[len(duration)-1:]

	num, err := strconv.Atoi(value)
	if err != nil || num < 0 {
		return 0, fmt.Errorf("invalid duration number: %s", value)
	}

	now := time.Now()
	var targetTime time.Time
	switch unit {
	case "m", "M":
		targetTime = now.AddDate(0, 0, 0-num*30) // 按30天计算一个月
	case "h", "H":
		targetTime = now.Add(-time.Duration(num) * time.Hour)
	case "d", "D":
		targetTime = now.AddDate(0, 0, -num)
	case "w", "W":
		targetTime = now.AddDate(0, 0, -num*7)
	case "y", "Y":
		targetTime = now.AddDate(-num, 0, 0)
	default:
		return 0, fmt.Errorf("unsupported duration unit: %s", unit)
	}

	return targetTime.UnixMilli(), nil
}
