package glog

import (
	"fmt"
	"msgPushSite/internal/glog/bot"
	"os"
	"path"
	"runtime"
	"strconv"

	. "msgPushSite/internal/glog/log"
)

func Info(args ...interface{}) {
	ZapLog.Named(funcName()).Info(args...)
}

func Infof(template string, args ...interface{}) {
	ZapLog.Named(funcName()).Infof(template, args...)
}

func Warn(args ...interface{}) {
	ZapLog.Named(funcName()).Warn(args...)
}

func Warnf(template string, args ...interface{}) {
	ZapLog.Named(funcName()).Warnf(template, args...)
}

func Error(args ...interface{}) {
	ZapLog.Named(funcName()).Error(args...)
}

func Debug(args ...interface{}) {
	ZapLog.Named(funcName()).Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	ZapLog.Named(funcName()).Debugf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	ZapLog.Named(funcName()).Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	ZapLog.Named(funcName()).Fatalf(template, args...)
}

func funcName() string {
	pc, _, _, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()
	return path.Base(funcName)
}

func lastIndexByte(s string, c byte) int {
	var count int
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			count++
		}

		if count == 2 {
			return i
		}
	}
	return -1
}

// 非常重要日志警告
func Emergency(template string, args ...interface{}) {
	s1, s2 := funcName4Emergency()
	bot.SendDefault(bot.SlackTemplate, s2, fmt.Sprintf(template, args...))
	ZapLog.Named(s1).Warnf(template, args...)
}

func funcName4Emergency() (string, string) {
	pc, f, line, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()

	//获取上一层的stack
	index := lastIndexByte(f, os.PathSeparator)
	if index != -1 {
		f = f[index+1:]
	}
	return path.Base(funcName),
		path.Base(funcName) + " " + f + ":" + strconv.Itoa(line) + " "
}
