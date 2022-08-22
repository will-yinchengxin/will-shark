package core

import (
	"errors"
	"fmt"
	"github.com/goinggo/mapstructure"
	"reflect"
)

func GetMapConfig(source map[string]interface{}, name string, obj interface{}) (map[string]interface{}, error) {
	if source == nil || len(source) < 1 {
		return nil, errors.New("source config is incorrect")
	}

	conf, ok := source[name]
	if conf == nil || !ok {
		str := fmt.Sprintf("config %s not found", name)
		return nil, errors.New(str)
	}

	confMap, ok := conf.(map[string]interface{})
	if !ok {
		str := fmt.Sprintf("config %s is incorrect", name)
		return nil, errors.New(str)
	}

	target := make(map[string]interface{})
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr { //指针类型获取真正type需要调用Elem
		t = t.Elem()
	}
	for name, val := range confMap {
		conn := reflect.New(t).Interface()
		if err := mapstructure.Decode(val, conn); err == nil {
			target[name] = conn
		} else {
			panic(err)
		}
	}

	return target, nil
}
func GetSingleConfig(source map[string]interface{}, name string, obj interface{}) (interface{}, error) {
	if source == nil || len(source) < 1 {
		return nil, errors.New("source config is incorrect")
	}

	conf, ok := source[name]
	if conf == nil || !ok {
		str := fmt.Sprintf("config %s not found", name)
		return nil, errors.New(str)
	}

	t := reflect.TypeOf(obj).Name()
	switch t {
	case "string":
		return conf.(string), nil
	case "int":
		return conf.(int), nil
	case "int64":
		return conf.(int64), nil
	case "uint":
		return conf.(uint), nil
	case "uint32":
		return conf.(uint32), nil
	case "uint64":
		return conf.(uint64), nil
	case "float32":
		return conf.(float32), nil
	case "float64":
		return conf.(float64), nil
	case "bool":
		return conf.(bool), nil
	default:
		t := reflect.TypeOf(obj)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		info := reflect.New(t).Interface()
		if err := mapstructure.Decode(conf, info); err == nil {
			return info, nil
		}

		str := fmt.Sprintf("config %s not incorrect", name)
		return nil, errors.New(str)
	}

}
