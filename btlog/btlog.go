package btlog

import (
	"fmt"
	"sync"
)

var ins *btLog
var once sync.Once

func Ins() *btLog {
	once.Do(func() {
		ins = &btLog{}
	})
	return ins
}

// todo 先直接打印
type btLog struct{}

func (btLog) Debug(v ...interface{}) {
	fmt.Println(v)
}

func (btLog) Info(v ...interface{}) {
	fmt.Println(v)
}

func (btLog) Warn(v ...interface{}) {
	fmt.Println(v)
}

func (btLog) Error(v ...interface{}) {
	fmt.Println(v)
}

func (btLog) Debugf(format string, v ...interface{}) {
	fmt.Println(format, v)
}

func (btLog) Infof(format string, v ...interface{}) {
	fmt.Println(format, v)
}

func (btLog) Warnf(format string, v ...interface{}) {
	fmt.Println(format, v)
}

func (btLog) Errorf(format string, v ...interface{}) {
	fmt.Println(format, v)
}
