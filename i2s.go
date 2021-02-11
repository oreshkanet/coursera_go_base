package main

import (
	"fmt"
	"reflect"
)

func i2s(data interface{}, out interface{}) error {
	valOut := reflect.ValueOf(out)
	if valOut.Kind() != reflect.Ptr {
		return fmt.Errorf("unknown type")
	}

	return i2sRecursive(data, valOut.Elem())
}

func i2sRecursive(data interface{}, out reflect.Value) error {
	switch out.Type().Kind() {
	case reflect.Struct:
		var dataTyped map[string]interface{}
		var isMap bool
		if dataTyped, isMap = data.(map[string]interface{}); !isMap {
			return fmt.Errorf("unknown type")
		}

		typeOut := out.Type()

		for i := 0; i < typeOut.NumField(); i++ {
			valueFieldOut := out.Field(i)
			typeFieldOut := typeOut.Field(i)

			if valData, isExists := dataTyped[typeFieldOut.Name]; isExists {
				if err := i2sRecursive(valData, valueFieldOut); err != nil {
					return err
				}
			}
		}
	case reflect.Slice:
		var dataTyped []interface{}
		var isMap bool
		if dataTyped, isMap = data.([]interface{}); !isMap {
			return fmt.Errorf("unknown type")
		}

		typeOut := out.Type().Elem()
		for _, valData := range dataTyped {
			newValOut := reflect.New(typeOut)
			if err := i2sRecursive(valData, newValOut.Elem()); err != nil {
				return err
			}
			out.Set(reflect.Append(out, newValOut.Elem()))
		}
	case reflect.Map:
		fmt.Println("map")
	case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
		if currentVal, err := i2int(data); err == nil {
			out.SetInt(currentVal)
		} else {
			return fmt.Errorf("unknown type")
		}
	case reflect.Float64, reflect.Float32:
		if currentVal, ok := data.(float64); ok {
			out.SetFloat(currentVal)
		} else {
			return fmt.Errorf("unknown type")
		}
	case reflect.String:
		if currentVal, ok := data.(string); ok {
			out.SetString(currentVal)
		} else {
			return fmt.Errorf("unknown type")
		}
	case reflect.Bool:
		if currentVal, ok := data.(bool); ok {
			out.SetBool(currentVal)
		} else {
			return fmt.Errorf("unknown type")
		}
	case reflect.Ptr:
		fmt.Println("Ptr")
		if out.CanAddr() {
			return i2sRecursive(data, out.Addr().Elem())
		} else {
			return i2sRecursive(data, out.Elem())
		}
	default:
		fmt.Println("unknown")
	}

	return nil
}

func i2int(data interface{}) (int64, error) {
	switch v := data.(type) {
	case int64, int32, int16, int8, int:
		return v.(int64), nil
	case float64, float32:
		return int64(v.(float64)), nil
	default:
		return 0, fmt.Errorf("unknown type of data to convert to int: %v", data)
	}
}
