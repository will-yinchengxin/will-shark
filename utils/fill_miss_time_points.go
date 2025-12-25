package utils

import (
	"fmt"
	"reflect"
)

func FillMissingTimePoints(data interface{}, startTime, endTime int64, interval int64) (interface{}, error) {
	if interval == 0 {
		return nil, fmt.Errorf("interval cannot be zero")
	}

	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf("data must be a slice")
	}
	if dataValue.Len() == 0 {
		return nil, nil
	}

	elemType := dataValue.Type().Elem()

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
		timePoint := int64(elem.Field(timePointFieldIndex).Uint())
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
			zeroElem.Field(timePointFieldIndex).SetUint(uint64(tp))
			resultSlice = reflect.Append(resultSlice, zeroElem)
		}
	}

	return resultSlice.Interface(), nil
}
