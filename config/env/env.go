package env

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const prefix = "COUPER_"

func Decode(conf interface{}) {
	envMap := make(map[string]string)
	for _, v := range os.Environ() {
		key := strings.Split(v, "=")
		if !strings.HasPrefix(key[0], prefix) {
			continue
		}
		envMap[strings.ToLower(key[0][len(prefix):])] = key[1]
	}

	if len(envMap) == 0 {
		return
	}

	val := reflect.ValueOf(conf)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)

		switch val.Field(i).Kind() {
		case reflect.Ptr:
			continue
		case reflect.Struct:
			Decode(val.Field(i).Interface())
		default:
		}

		envVal, ok := field.Tag.Lookup("env")
		if !ok {
			continue
		}

		mapVal, exist := envMap[envVal]
		if !exist {
			continue
		}

		switch val.Field(i).Interface().(type) {
		case bool:
			val.Field(i).SetBool(mapVal == "true")
		case int:
			intVal, err := strconv.Atoi(mapVal)
			if err != nil {
				panic(err)
			}
			val.Field(i).SetInt(int64(intVal))
		case string:
			val.Field(i).SetString(mapVal)
		default:
			panic(fmt.Sprintf("type mapping not implemented: %v", reflect.TypeOf(val.Field(i).Interface())))
		}
	}
}
