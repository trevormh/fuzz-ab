package main

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

func ReadFile(path string) ([]byte,error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return b, err
}

func CopyMap(m map[string]interface{}) map[string]interface{} {
    cp := make(map[string]interface{})
    for k, v := range m {
        vm, ok := v.(map[string]interface{})
        if ok {
            cp[k] = CopyMap(vm)
        } else {
            cp[k] = v
        }
    }
    return cp
}

func ConvertToStr(val interface{}) (string) {
	var str_val string
	if str, ok := val.(string); ok {
		str_val = str
	} else if float_val, ok := val.(float64); ok {
		str_val = strconv.FormatFloat(float_val, 'f', -1, 64)
	} else if intval, ok := val.(int); ok {
		return strconv.Itoa(intval)
	} else {
		fmt.Println(val)
		fmt.Println(reflect.TypeOf(val))
		panic("Value could not be cast to string")
	}
	return str_val
}


func ConvertToInt(val interface{}) (int) {
	var int_val int
	if str, ok := val.(string); ok {
		res, err := strconv.Atoi(str)
		if err != nil {
			panic("Unable to convert to int")
		}
		int_val = res
	} else {
		fmt.Println(val)
		fmt.Println(reflect.TypeOf(val))
		panic("Value could not be cast to string")
	}
	return int_val
}