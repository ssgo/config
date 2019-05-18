package config

import (
	"encoding/json"
	errors2 "errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"os/user"
	"reflect"
	"strings"
)

type openStatusType int

const NONE openStatusType = 0
const JSON openStatusType = 1
const YML openStatusType = 2

var envConfigs = map[string]string{}
var envUpperConfigs = map[string]string{}
var inited = false

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

func LoadConfig(name string, conf interface{}) []error {
	if !inited {
		inited = true
		initConfig()
	}

	var file *os.File
	openStatus := NONE
	lenOsArgs := len(os.Args)
	if lenOsArgs >= 1 {
		execPath := os.Args[0]
		pos := strings.LastIndex(os.Args[0], string(os.PathSeparator))
		if pos != -1 {
			execPath = os.Args[0][0:pos]
		}
		//file, err = os.Open(execPath + "/" + name + ".json")
		file, openStatus = openFile(execPath + "/" + name)
	}
	//if err != nil || lenOsArgs < 1 {
	if file == nil || lenOsArgs < 1 {
		//file, err = os.Open(name + ".json")
		file, openStatus = openFile(name)
		//if err != nil {
		if file == nil {
			//file, err = os.Open("../" + name + ".json")
			file, openStatus = openFile("../" + name)
			//if err != nil {
			if file == nil {
				u, _ := user.Current()
				if u != nil {
					//file, err = os.Open(u.HomeDir + "/" + name + ".json")
					file, openStatus = openFile(u.HomeDir + "/" + name)
				}
			}
		}
	}

	errors := make([]error, 0)
	if file != nil {
		if openStatus == YML {
			decoder := yaml.NewDecoder(file)
			err := decoder.Decode(conf)
			if err != nil {
				errors = append(errors, err)
			}
			_ = file.Close()
			//fmt.Println(file.Name(), conf)
		} else {
			decoder := json.NewDecoder(file)
			err := decoder.Decode(conf)
			if err != nil {
				errors = append(errors, err)
			}
			_ = file.Close()
		}
	}

	makeEnvConfig(name, reflect.ValueOf(conf), &errors)
	//b, _ := json.MarshalIndent(conf, "", "  ")
	//fmt.Println(string(b))

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
	if ev != "" {
		//fmt.Println("    ^^^^^^^", prefix, v.CanSet())
		newValue := reflect.New(t)
		var resultValue reflect.Value
		err := json.Unmarshal([]byte(ev), newValue.Interface())
		if err != nil && t.Kind() == reflect.String {
			//v.SetString(ev)
			resultValue = reflect.ValueOf(ev)
		} else if err == nil {
			//v.Set(newValue.Elem())
			resultValue = newValue.Elem()
		} else {
			*errors = append(*errors, errors2.New(fmt.Sprint(err.Error(), ", prefix:", prefix, ", event:", ev)))
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
			if v.Field(i).Kind() == reflect.Ptr && v.Field(i).IsNil() {
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
				if strings.HasPrefix(k1, findPrefix) || strings.HasPrefix(strings.ToUpper(k1), strings.ToUpper(findPrefix)) {
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
						v.SetMapIndex(reflect.ValueOf(k2), v1)
					}
				}
			}
		}
		for _, mk := range v.MapKeys() {
			//fmt.Println("    ^^^^^^^", prefix+"_"+toString(mk), mk, v.MapIndex(mk))
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
	if t.Kind() == reflect.Interface {
		t = reflect.TypeOf(v.Interface())
		v = reflect.ValueOf(v.Interface())
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

func openFile(filePrefix string) (*os.File, openStatusType) {

	fi, err := os.Stat(filePrefix + ".yml")
	if err == nil && fi != nil {
		file, err := os.Open(filePrefix + ".yml")
		if err == nil && file != nil {
			return file, YML
		}
	}

	fi, err = os.Stat(filePrefix + ".json")
	if err == nil && fi != nil {
		file, err := os.Open(filePrefix + ".json")
		if err == nil && file != nil {
			return file, JSON
		}
	}

	return nil, NONE
}
