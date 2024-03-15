package utils

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"
)

var (
	BeforeTime = "2024-03-01 00:00:00"
	timeErr    = errors.New("时间格式错误,注意检查 !!!")
)

type TimeInterval struct {
	StartTime time.Time
	EndTime   time.Time
}

func GetTimeSeries(timeStrArr [][]string) ([]TimeInterval, error) {
	timeSeries := make([]TimeInterval, 0, len(timeStrArr))
	re := regexp.MustCompile("^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}$")
	for key, _ := range timeStrArr {
		matchStart := re.MatchString(timeStrArr[key][0])
		if !matchStart {
			return nil, timeErr
		}
		matchEnd := re.MatchString(timeStrArr[key][1])
		if !matchEnd {
			return nil, timeErr
		}
		tmp := TimeInterval{
			StartTime: parseTime(timeStrArr[key][0]),
			EndTime:   parseTime(timeStrArr[key][1]),
		}
		timeSeries = append(timeSeries, tmp)
	}
	err := CheckRange(timeSeries)
	if err != nil {
		return nil, err
	}
	sort.Slice(timeSeries, func(i, j int) bool {
		return timeSeries[i].StartTime.Before(timeSeries[j].StartTime)
	})
	err = CheckOverlap(timeSeries)
	if err != nil {
		return nil, err
	}

	return timeSeries, nil
}

func CheckOverlap(timeSeries []TimeInterval) error {
	for i := 0; i < len(timeSeries)-1; i++ {
		if timeSeries[i].EndTime.After(timeSeries[i+1].StartTime) {
			fmt.Println("Error: 时间序列存在重叠")
			return errors.New("时间序列存在重叠")
		}
	}
	return nil
}

func CheckRange(timeSeries []TimeInterval) error {
	for _, interval := range timeSeries {
		if interval.StartTime.Before(parseTime(BeforeTime)) || interval.EndTime.Before(parseTime(BeforeTime)) {
			fmt.Println("Error: 时间格式不正确或超出范围")
			return errors.New("时间范围不正确")
		}
	}
	return nil
}

func parseTime(timeStr string) time.Time {
	// 正则校验时间格式是否为 2006-01-02 15:04:05

	layout := GoBronTime
	t, err := time.Parse(layout, timeStr)
	if err != nil {
		return time.Time{}
	}
	return t
}
