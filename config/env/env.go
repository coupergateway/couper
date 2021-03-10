package env

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const PREFIX = "COUPER_"

var OsEnviron = os.Environ

func Decode(conf interface{}) {
	DecodeWithPrefix(conf, "")
}

func DecodeWithPrefix(conf interface{}, prefix string) {
	ctxPrefix := PREFIX + prefix
	envMap := make(map[string]string)
	for _, v := range OsEnviron() {
		key := strings.Split(v, "=")
		if !strings.HasPrefix(key[0], ctxPrefix) {
			continue
		}
		envMap[strings.ToLower(key[0][len(ctxPrefix):])] = key[1]
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
			DecodeWithPrefix(val.Field(i).Interface(), prefix)
		default:
		}

		envVal, ok := field.Tag.Lookup("env")
		if !ok { // fallback to hcl struct tag
			envVal, ok = field.Tag.Lookup("hcl")
			if !ok {
				continue
			}
		}
		envVal = strings.Split(envVal, ",")[0]

		mapVal, exist := envMap[envVal]
		if !exist || mapVal == "" {
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
		case []string:
			val.Field(i).Set(reflect.ValueOf(strings.Split(mapVal, ",")))
		case time.Duration:
			parsedDuration, err := time.ParseDuration(mapVal)
			if err == nil {
				val.Field(i).Set(reflect.ValueOf(parsedDuration))
			}
		default:
			panic(fmt.Sprintf("env decode: type mapping not implemented: %v", reflect.TypeOf(val.Field(i).Interface())))
		}
	}
}
