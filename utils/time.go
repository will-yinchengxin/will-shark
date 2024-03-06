package utils

import "time"

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
