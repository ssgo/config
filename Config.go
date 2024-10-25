package config

import (
	"bytes"
	"encoding/json"
	errors2 "errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/ssgo/u"
)

//type openStatusType int
//
//const NONE openStatusType = 0
//const JSON openStatusType = 1
//const YML openStatusType = 2

var envConfigs = map[string]string{}
var envUpperConfigs = map[string]string{}
var inited = false

type Duration time.Duration

func (tm *Duration) ConfigureBy(setting string) {
	result, err := time.ParseDuration(setting)
	if err == nil {
		*tm = Duration(result)
	}
}

func (tm *Duration) MarshalJSON() ([]byte, error) {
	return []byte(time.Duration(*tm).String()), nil
}

func (tm *Duration) UnmarshalJSON(value []byte) error {
	result, err := time.ParseDuration(string(bytes.Trim(value, "\"")))
	if err == nil {
		*tm = Duration(result)
	}
	return err
}

func (tm *Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(*tm).String(), nil
}

func (tm *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	value := ""
	err := unmarshal(&value)
	if err != nil {
		return err
	}
	result, err := time.ParseDuration(value)
	if err == nil {
		*tm = Duration(result)
	}
	return err
}

func (tm *Duration) TimeDuration() time.Duration {
	return time.Duration(*tm)
}

type Configurable interface {
	ConfigureBy(setting string)
}

func initConfig() {
	envConf := map[string]interface{}{}
	LoadConfig("env", &envConf)
	initEnvConfigFromFile("", reflect.ValueOf(envConf))

	for _, e := range os.Environ() {
		a := strings.SplitN(e, "=", 2)
		if len(a) == 2 {
			envConfigs[a[0]] = a[1]
		}
	}
	for k1, v1 := range envConfigs {
		envUpperConfigs[strings.ToUpper(k1)] = v1
	}

	//b, _ := json.MarshalIndent(envUpperConfigs, "", "  ")
	//fmt.Println(string(b))
}

func ResetConfigEnv() {
	envConfigs = map[string]string{}
	envUpperConfigs = map[string]string{}
	initConfig()
}

func searchFile(checkPath, name string, searched *map[string]bool) string {
	for {
		if !(*searched)[checkPath] {
			(*searched)[checkPath] = true
			if filename := checkFile(filepath.Join(checkPath, name)); filename != "" {
				return filename
			}
		}
		oldPath := checkPath
		checkPath = filepath.Dir(oldPath)
		if oldPath == checkPath {
			return ""
		}
	}
}

func LoadConfig(name string, conf interface{}) []error {
	if !inited {
		inited = true
		initConfig()
	}

	searched := map[string]bool{}
	// search current path
	currentPath, _ := os.Getwd()
	filename := searchFile(currentPath, name, &searched)
	if filename == "" {
		// search exec path
		execPath, _ := filepath.Abs(os.Args[0])
		filename = searchFile(filepath.Dir(execPath), name, &searched)
		if filename == "" {
			// search user home path
			homePath, _ := os.UserHomeDir()
			filename = checkFile(filepath.Join(homePath, name))
		}
	}

	errors := make([]error, 0)
	if filename != "" {
		if err := u.LoadX(filename, conf); err != nil {
			errors = append(errors, err)
		}
	}

	makeEnvConfig(name, reflect.ValueOf(conf), &errors)

	if len(errors) == 0 {
		return nil
	}
	return errors
}

func makeEnvConfig(prefix string, v reflect.Value, errors *[]error) *reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Invalid {
		return nil
	}

	t := v.Type()
	ev := envConfigs[prefix]
	if ev == "" {
		ev = envUpperConfigs[strings.ToUpper(prefix)]
	}
	//fmt.Println("    ^^^^^^^", prefix, ev)

	if ev != "" {
		//fmt.Println("    ^^^^^^^1", prefix, v.CanSet(), v.CanAddr())
		newValue := reflect.New(t)
		var resultValue reflect.Value

		//injectObjValue := reflect.ValueOf(injectObj)
		//setLoggerMethod, found := injectObjValue.Type().MethodByName("ConfigureBy")
		//if found && setLoggerMethod.Type.NumIn() == 2 && setLoggerMethod.Type.In(1).String() == "*log.Logger" {
		//	setLoggerMethod.Func.Call([]reflect.Value{injectObjValue, reflect.ValueOf(requestLogger)})
		//}

		foundConfigureBy := false
		if v.CanAddr() {
			configureMethod, found := v.Addr().Type().MethodByName("ConfigureBy")
			//fmt.Println("      ^^^^^^^2", prefix, v.Type().String(), found, configureMethod)
			if !strings.HasPrefix(ev, "{") && found && configureMethod.Type.NumIn() == 2 && configureMethod.Type.In(1).Kind() == reflect.String {
				//fmt.Println("      ^^^^^^^2", prefix, v.CanSet(), v.CanAddr())
				configureMethod.Func.Call([]reflect.Value{v.Addr(), reflect.ValueOf(ev)})
				foundConfigureBy = found
			}
		}
		if !foundConfigureBy {
			//fmt.Println("      ^^^^^^^3", prefix, v.CanSet(), v.CanAddr())
			err := json.Unmarshal([]byte(ev), newValue.Interface())
			//fmt.Println("      ^^^^^^^4", err)
			if err != nil && t.Kind() == reflect.String {
				//v.SetString(ev)
				resultValue = reflect.ValueOf(ev)
			} else if err == nil {
				//v.Set(newValue.Elem())
				resultValue = newValue.Elem()
			} else {
				*errors = append(*errors, errors2.New(fmt.Sprint(err.Error(), ", prefix:", prefix, ", event:", ev)))
			}
		}

		if !resultValue.IsValid() {
			return nil
		}

		if v.CanSet() {
			v.Set(resultValue)
			return nil
		} else {
			return &resultValue
		}

		//if v.CanSet() {
		//	newValue := reflect.New(t)
		//	err := json.Unmarshal([]byte(ev), newValue.Interface())
		//	if err != nil && t.Kind() == reflect.String {
		//		v.SetString(ev)
		//	} else if err == nil {
		//		v.Set(newValue.Elem())
		//	} else {
		//		*errors = append(*errors, errors2.New(fmt.Sprint(err.Error(), ", prefix:", prefix, ", event:", ev)))
		//	}
		//} else {
		//	*errors = append(*errors, errors2.New(fmt.Sprint("Can't set config because CanSet() == false",
		//		", prefix:", prefix,
		//		", event:", ev,
		//		", varType:", fmt.Sprint(t),
		//		", value:", toString(v))))
		//}
	}

	if t.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).Name[0] > 90 {
				continue
			}
			if v.Field(i).Kind() == reflect.Ptr && v.Field(i).IsNil() {
				//fmt.Println("      ^^^^^^^1", prefix, v.Type().Field(i).Name, v.Field(i).String())
				v.Field(i).Set(reflect.New(v.Field(i).Type().Elem()))
			}
			resultValue := makeEnvConfig(prefix+"_"+v.Type().Field(i).Name, v.Field(i), errors)
			if resultValue != nil && v.Field(i).CanSet() {
				v.Field(i).Set(*resultValue)
			}
		}
	} else if t.Kind() == reflect.Map {
		// 查找 环境变量 或 env.json 中是否有配置项
		if t.Elem().Kind() != reflect.Interface {
			findPrefix := prefix + "_"
			for k1 := range envConfigs {
				//fmt.Println("      ^^^^^^^ Map", prefix, k1)
				if strings.HasPrefix(k1, findPrefix) || strings.HasPrefix(strings.ToUpper(k1), strings.ToUpper(findPrefix)) {
					//fmt.Println("      ^^^^^^^ Map", prefix, k1)
					findPostfix := k1[len(findPrefix):]
					a1 := strings.Split(findPostfix, "_")
					k2 := ""
					if len(a1) > 0 {
						//k2 := strings.ToLower(a1[0])
						k2 = a1[0]
					}
					if k2 != "" && v.MapIndex(reflect.ValueOf(k2)).Kind() == reflect.Invalid {
						var v1 reflect.Value
						if t.Elem().Kind() == reflect.Ptr {
							v1 = reflect.New(t.Elem().Elem())
						} else {
							v1 = reflect.New(t.Elem()).Elem()
						}
						if len(v.MapKeys()) == 0 {
							v.Set(reflect.MakeMap(t))
						}
						//fmt.Println("        ^^^^^^^ Map", prefix, k1, k2)
						v.SetMapIndex(reflect.ValueOf(k2), v1)
					}
				}
			}
		}
		for _, mk := range v.MapKeys() {
			resultValue := makeEnvConfig(prefix+"_"+toString(mk), v.MapIndex(mk), errors)
			if resultValue != nil {
				v.SetMapIndex(mk, *resultValue)
			}
		}
	} else if t.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			if v.Index(i).Kind() == reflect.Ptr && v.Index(i).IsNil() {
				v.Index(i).Set(reflect.New(v.Index(i).Type().Elem()))
			}
			resultValue := makeEnvConfig(fmt.Sprint(prefix, "_", i), v.Index(i), errors)
			if resultValue != nil && v.Index(i).CanSet() {
				v.Index(i).Set(*resultValue)
			}
		}
	}

	return nil
}

func toString(v reflect.Value) string {
	if v.Kind() == reflect.String {
		return v.String()
	} else {
		return fmt.Sprint(v)
	}
}

func initEnvConfigFromFile(prefix string, v reflect.Value) {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	if t == nil {
		return
	}
	if t.Kind() == reflect.Interface {
		t = reflect.TypeOf(v.Interface())
		v = reflect.ValueOf(v.Interface())
	}
	if t == nil {
		return
	}
	if t.Kind() == reflect.Map {
		if prefix != "" {
			prefix += "_"
		}
		for _, mk := range v.MapKeys() {
			initEnvConfigFromFile(prefix+toString(mk), v.MapIndex(mk))
		}
	} else if t.Kind() == reflect.String {
		envConfigs[prefix] = v.String()
	} else {
		b, err := json.Marshal(v.Interface())
		if err == nil {
			envConfigs[prefix] = string(b)
		} else {
			envConfigs[prefix] = fmt.Sprint(v.Interface())
		}
	}
}

func checkFile(filePrefix string) string {
	if u.FileExists(filePrefix + ".yml") {
		return filePrefix + ".yml"
	} else if u.FileExists(filePrefix + ".json") {
		return filePrefix + ".json"
	}
	return ""
}

//func openFile(filePrefix string) (*os.File, openStatusType) {
//
//	fi, err := os.Stat(filePrefix + ".yml")
//	if err == nil && fi != nil {
//		file, err := os.Open(filePrefix + ".yml")
//		if err == nil && file != nil {
//			return file, YML
//		}
//	}
//
//	fi, err = os.Stat(filePrefix + ".json")
//	if err == nil && fi != nil {
//		file, err := os.Open(filePrefix + ".json")
//		if err == nil && file != nil {
//			return file, JSON
//		}
//	}
//
//	return nil, NONE
//}
