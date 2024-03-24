package config

import (
	"github.com/ssgo/u"
	"os"
	"testing"
	"time"
)

type TestList struct {
	Name string
}

func (l *TestList) ConfigureBy(setting string) {
	l.Name = setting
}

func TestForMap(t *testing.T) {
	os.Setenv("test_list_ccc", "333")
	os.Setenv("test_list_ddd", "{\"name\":\"444\"}")
	testConf := map[string]interface{}{}
	err := LoadConfig("test", &testConf)
	if err != nil {
		t.Error("read test.json failed", err)
	}
	if testConf["name"] != "test-config" {
		t.Error("name in test.json failed", testConf["name"])
	}
}

type testConfType struct {
	Name     string
	Sets     []int
	List     map[string]*TestList
	List2    []string
	Duration Duration
}

func TestForStruct(t *testing.T) {
	testConf := testConfType{}
	_ = LoadConfig("test", &testConf)
	if testConf.Name != "test-config" {
		t.Error("name in test.json failed", testConf.Name)
	}
	if len(testConf.Sets) != 3 || testConf.Sets[1] != 2 {
		t.Error("sets in test.json failed", testConf.Sets)
	}
	if testConf.List == nil || testConf.List["aaa"].Name != "222" {
		t.Error("aaa in test.json failed", testConf.List["aaa"].Name)
	}
	if testConf.List == nil || (testConf.List["bbb"] == nil || testConf.List["bbb"].Name != "xxx") {
		t.Error("bbb in env.json failed", testConf.List, testConf.List["bbb"])
	}
	if testConf.List == nil || testConf.List["ccc"] == nil || testConf.List["ccc"].Name != "333" {
		t.Error("ccc in test.json failed", testConf.List["ccc"])
	}
	if testConf.List == nil || testConf.List["ddd"] == nil || testConf.List["ddd"].Name != "444" {
		t.Error("ddd in test.json failed", testConf.List["ddd"])
	}
	if time.Duration(testConf.Duration) != 100*time.Second {
		t.Error("time in test.json failed", testConf.Duration.TimeDuration())
	}
	if time.Duration(testConf.Duration).String() != (100 * time.Second).String() {
		t.Error("time in test.json failed", testConf.Duration.TimeDuration(), time.Duration(testConf.Duration).String())
	}
}

func TestForYml(t *testing.T) {
	testConf := testConfType{}
	_ = LoadConfig("test2", &testConf)
	if testConf.Name != "test-config" {
		t.Error("name in test.yml failed", testConf.Name)
	}
	if len(testConf.Sets) != 3 || testConf.Sets[1] != 2 {
		t.Error("sets in test.yml failed", testConf.Sets)
	}
	if testConf.List == nil || testConf.List["aaa"].Name != "222" {
		t.Error("map in test.yml failed", testConf.List["aaa"])
	}
	if testConf.List2 == nil || len(testConf.List2) != 2 {
		t.Error("list2 in test.yml failed", testConf.List["aaa"])
	}
	if testConf.List != nil && (testConf.List["bbb"] == nil || testConf.List["bbb"].Name != "xxx") {
		t.Error("map in env.yml failed", testConf.List, testConf.List["bbb"])
	}
}

func TestForMemFile(t *testing.T) {
	u.AddFileToMemory(u.MemFile{
		Name:    "test3.yml",
		ModTime: time.Now(),
		IsDir:   false,
		Data: []byte(`name: test-config
list:
  aaa:
    name: 111
list2:
  - ls -l /
  - cp a b
`)})
	u.AddFileToMemory(u.MemFile{
		Name:    "env.yml",
		ModTime: time.Now(),
		IsDir:   false,
		Data: []byte(`test:
  list:
    aaa:
      name: "222"
    bbb:
      name: xxx
  list2:
    aaa: 111

TEST_SETS:
  - 1
  - 2
  - 3

test2:
  list:
    aaa:
      name: "222"
    bbb:
      name: xxx
  list2:
    aaa: 111

TEST2_SETS:
  - 1
  - 2
  - 3

test3:
  list:
    aaa:
      name: "222"
    bbb:
      name: xxx
  list2:
    aaa: 111

TEST3_SETS:
  - 1
  - 2
  - 3
`)})
	ResetConfigEnv()
	testConf := testConfType{}
	_ = LoadConfig("test3", &testConf)
	if testConf.Name != "test-config" {
		t.Error("name in test.yml failed", testConf.Name)
	}
	if len(testConf.Sets) != 3 || testConf.Sets[1] != 2 {
		t.Error("sets in test.yml failed", testConf.Sets)
	}
	if testConf.List == nil || testConf.List["aaa"].Name != "222" {
		t.Error("map in test.yml failed", testConf.List["aaa"])
	}
	if testConf.List2 == nil || len(testConf.List2) != 2 {
		t.Error("list2 in test.yml failed", testConf.List["aaa"])
	}
	if testConf.List != nil && (testConf.List["bbb"] == nil || testConf.List["bbb"].Name != "xxx") {
		t.Error("map in env.yml failed", testConf.List, testConf.List["bbb"])
	}
}
