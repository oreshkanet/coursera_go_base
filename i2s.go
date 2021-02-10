package main

import (
	"fmt"
	"reflect"
)

func i2s(data interface{}, out interface{}) error {

	// Будем итерироваться по элементам out
	valOut := reflect.ValueOf(out).Elem()
	typeOut := reflect.TypeOf(out).Elem()
	for i := 0; i < typeOut.NumField(); i++ {
		valueFieldOut := valOut.Field(i)
		typeFieldOut := typeOut.Field(i)

		// В зависимости от типа data будет разная логика чтения данных
		switch dataTyped := data.(type) {
		// Чтение из MAP
		case map[string]interface{}:
			if valData, isExists := dataTyped[typeFieldOut.Name]; isExists {
				switch typeFieldOut.Type.Kind() {
				case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
					if currentVal, err := i2int(valData); err == nil {
						valueFieldOut.SetInt(currentVal)
					}
				case reflect.Float64, reflect.Float32:
					valueFieldOut.SetFloat(valData.(float64))
				case reflect.String:
					valueFieldOut.SetString(valData.(string))
				case reflect.Bool:
					valueFieldOut.SetBool(valData.(bool))
				case reflect.Struct:
					curValue := reflect.New(typeFieldOut.Type) //valueFieldOut.Interface()
					i2struct(valData, valueFieldOut.Addr())
					fmt.Println(curValue)
				default:

					fmt.Printf("\tname=%v, type=%v, value=%v, tag=`%v`\n", typeFieldOut.Name,
						typeFieldOut.Type.Kind(),
						valueFieldOut,
						typeFieldOut.Tag,
					)
					//i2s(valData, typeFieldOut.Interface())
				}
			}

		default:
			return fmt.Errorf("unknown type of data")
		}

	}

	return nil
}

func i2struct(data interface{}, valOut reflect.Value) error {

	// Будем итерироваться по элементам out
	//valOut := reflect.ValueOf(out).Elem()
	typeOut := valOut.Type() //reflect.TypeOf(out).Elem()
	for i := 0; i < valOut.NumField(); i++ {
		valueFieldOut := valOut.Field(i)
		typeFieldOut := typeOut.Field(i)

		// В зависимости от типа data будет разная логика чтения данных
		switch dataTyped := data.(type) {
		// Чтение из MAP
		case map[string]interface{}:
			if valData, isExists := dataTyped[typeFieldOut.Name]; isExists {
				switch typeFieldOut.Type.Kind() {
				case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
					if currentVal, err := i2int(valData); err == nil {
						valueFieldOut.SetInt(currentVal)
					}
				case reflect.Float64, reflect.Float32:
					valueFieldOut.SetFloat(valData.(float64))
				case reflect.String:
					valueFieldOut.SetString(valData.(string))
				case reflect.Bool:
					valueFieldOut.SetBool(valData.(bool))
				case reflect.Struct:
					curValue := reflect.New(typeFieldOut.Type) //valueFieldOut.Interface()
					i2s(valData, valueFieldOut)
					fmt.Println(curValue)
				default:

					fmt.Printf("\tname=%v, type=%v, value=%v, tag=`%v`\n", typeFieldOut.Name,
						typeFieldOut.Type.Kind(),
						valueFieldOut,
						typeFieldOut.Tag,
					)
					//i2s(valData, typeFieldOut.Interface())
				}
			}

		default:
			return fmt.Errorf("unknown type of data")
		}

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
