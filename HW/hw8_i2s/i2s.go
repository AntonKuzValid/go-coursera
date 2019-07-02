package main

import (
	"fmt"
	"reflect"
)

func i2s(data interface{}, out interface{}) (err error) {

	ref := reflect.ValueOf(out)
	if ref.Kind() != reflect.Ptr {
		return fmt.Errorf("err")
	}

	valOut := ref.Elem()
	switch valOut.Kind() {
	case reflect.Struct:
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("err")
		}
		for i := 0; i < valOut.NumField(); i++ {
			field := valOut.Field(i)
			valType := valOut.Type().Field(i)
			val := dataMap[valType.Name]
			switch field.Type().Kind() {
			case reflect.Int:
				valFloat, ok := val.(float64)
				if !ok {
					return fmt.Errorf("err")
				}
				valInt := int(valFloat)
				field.Set(reflect.ValueOf(valInt))
			case reflect.String:
				valStr, ok := val.(string)
				if !ok {
					return fmt.Errorf("err")
				}
				field.SetString(valStr)
			case reflect.Bool:
				valBool, ok := val.(bool)
				if !ok {
					return fmt.Errorf("err")
				}
				field.SetBool(valBool)
			case reflect.Struct, reflect.Slice:
				if err = i2s(val, field.Addr().Interface()); err != nil {
					return
				}
			default:
				return fmt.Errorf("unsupported %v", field.Type().Kind())

			}
		}
	case reflect.Slice:
		valSlice, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("err")
		}
		sliceLen := len(valSlice)
		valOut.Set(reflect.MakeSlice(valOut.Type(), sliceLen, sliceLen))
		for ind := range valSlice {
			if err = i2s(valSlice[ind], valOut.Index(ind).Addr().Interface()); err != nil {
				return
			}
		}
	default:
		return fmt.Errorf("err")
	}

	return
}
