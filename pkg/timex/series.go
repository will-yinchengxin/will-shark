package timex

import (
	"fmt"
	"math"
	"reflect"
)

func FillMissingTimePoints(data interface{}, startTime, endTime int64, interval int64) (interface{}, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("interval must be positive")
	}
	// TimePoint 为 uint32，超出范围的时间点会被截断
	if startTime < 0 || endTime > math.MaxUint32+1 {
		return nil, fmt.Errorf("time range must be within [0, %d]", uint64(math.MaxUint32))
	}

	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf("data must be a slice")
	}
	if dataValue.Len() == 0 {
		return nil, nil
	}

	elemType := dataValue.Type().Elem()
	if elemType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("data must be a slice of struct")
	}

	timePointFieldIndex := -1
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if field.Name == "TimePoint" && field.Type.Kind() == reflect.Uint32 {
			timePointFieldIndex = i
			break
		}
	}

	if timePointFieldIndex == -1 {
		return nil, fmt.Errorf("struct must have a TimePoint field of type uint32")
	}

	dataMap := make(map[int64]reflect.Value)
	for i := 0; i < dataValue.Len(); i++ {
		elem := dataValue.Index(i)
		timePoint := int64(elem.Field(timePointFieldIndex).Uint()) // #nosec G115 -- 源字段为 uint32，转换为 int64 不会溢出
		dataMap[timePoint] = elem
	}

	var timePoints []int64
	for t := startTime; t < endTime; t += interval {
		timePoints = append(timePoints, t)
	}

	resultSlice := reflect.MakeSlice(dataValue.Type(), 0, len(timePoints))

	for _, tp := range timePoints {
		if existingData, exists := dataMap[tp]; exists {
			resultSlice = reflect.Append(resultSlice, existingData)
		} else {
			zeroElem := reflect.New(elemType).Elem()
			zeroElem.Field(timePointFieldIndex).SetUint(uint64(tp)) // #nosec G115 -- 函数开头已校验时间范围在 [0, MaxUint32] 内
			resultSlice = reflect.Append(resultSlice, zeroElem)
		}
	}

	return resultSlice.Interface(), nil
}
