package utils

import (
	"fmt"
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

func ParseTimeStrToInt(timeStr string) int64 {
	parseTime, _ := time.ParseInLocation(GoBronTime, timeStr, time.Local)
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
	bDate, _ := time.ParseInLocation(format, beginDate, time.Local)
	eDate, _ := time.ParseInLocation(format, endDate, time.Local)
	day := int(eDate.Sub(bDate).Hours() / 24)
	dlist := make([]string, 0)
	dlist = append(dlist, beginDate)
	for i := 1; i < day; i++ {
		result := bDate.AddDate(0, 0, i)
		dlist = append(dlist, result.Format(format))
	}
	dlist = append(dlist, endDate)
	return dlist
}

// ParseDuration 解析时间字符串，返回对应的时间戳
func ParseDuration(duration string) (int64, error) {
	duration = strings.ToLower(duration)
	if len(duration) < 2 {
		return 0, fmt.Errorf("invalid duration format: %s", duration)
	}

	value := duration[:len(duration)-1]
	unit := duration[len(duration)-1:]

	var num int
	_, err := fmt.Sscanf(value, "%d", &num)
	if err != nil {
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
