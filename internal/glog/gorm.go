package glog

import (
	ctx "context"
	"fmt"
	"gorm.io/gorm"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm/logger"
)

type gormZapLogger struct {
}

func NewGormZapLogger() *gormZapLogger {
	return &gormZapLogger{}
}

func (gormZapLogger) Printf(prefix string, v ...interface{}) {
	if len(v) == 4 {
		Infof("[gorm-sql]:time=%v | row=%v | pos=%v |sql=%v", v[1], v[2], v[0], v[3])
		return
	}
	if len(v) == 5 {
		Infof("[gorm-sql]:time=%v | row=%v | pos=%v | sql=%v | err=%v", v[2], v[3], v[0], v[4], v[1])
		return
	}
	Infof("[gorm-sql]:%v", v)
}

type dbLog struct {
	logger.Config
	traceSwitch bool
}

func NewDBLog(config logger.Config) *dbLog {
	return &dbLog{Config: config, traceSwitch: true}
}

func (l *dbLog) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	newLogger.traceSwitch = true
	return &newLogger
}

func (l *dbLog) Info(ctx ctx.Context, s string, i ...interface{}) {
	if l.LogLevel >= logger.Info {
		Info(fmt.Sprintf("DBLOG |"+s+"\n", i...))
	}
}

func (l *dbLog) Warn(ctx ctx.Context, s string, i ...interface{}) {
	if l.LogLevel >= logger.Warn {
		Info(fmt.Sprintf("DBLOG |"+s+"\n", i...))
	}
}

func (l *dbLog) Error(ctx ctx.Context, s string, i ...interface{}) {
	if l.LogLevel >= logger.Error {
		Info(fmt.Sprintf("DBLOG |"+s+"\n", i...))
	}
}

func (l *dbLog) Trace(ctx ctx.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= logger.Silent || !l.traceSwitch {
		return
	}

	//耗时
	timestamp := time.Since(begin)

	sql, rows := fc()

	//如果报错直接打印日志
	if err != nil && err != gorm.ErrRecordNotFound {
		Emergency(
			"DBLOG |pos=%s |err=%v |sql=%s |rows=%d |timestamp=%.2fs",
			funcName4Gorm(),
			err,
			sql,
			rows,
			timestamp.Seconds(),
		)
		return
	}

	//产生慢查询
	if timestamp > l.SlowThreshold {

		//记录不同标准的慢sql
		var standard = "PURPLE"
		switch {
		case timestamp > l.SlowThreshold*8:
			standard = "RED"
		case timestamp > l.SlowThreshold*4:
			standard = "ORANGE"
		case timestamp > l.SlowThreshold*2:
			standard = "YELLOW"
		}

		Warnf(
			"DBLOG |SLOW |pos=%s |standard=%s |sql=%s |rows=%d |timestamp=%.2fs",
			funcName4Gorm(),
			standard,
			sql,
			rows,
			timestamp.Seconds(),
		)

		return
	}

	//如果不是慢日志，只有数据变更操作的日志才记录
	if rows > 0 && (strings.Contains(sql, "INSERT") || strings.Contains(sql, "UPDATE")) {
		Infof(
			"DBLOG |pos=%s|sql=%s |rows=%d |timestamp=%.2fs",
			funcName4Gorm(),
			sql,
			rows,
			timestamp.Seconds(),
		)
		return
	}

	//更新0的日志额外重点记录
	if strings.Contains(sql, "UPDATE") {
		Warnf(
			"DBLOG |pos=%s |sql=%s |rows=%d |timestamp=%.2fs",
			funcName4Gorm(),
			sql,
			rows,
			timestamp.Seconds(),
		)
		return
	}
}

func funcName4Gorm() string {
	pc, f, line, _ := runtime.Caller(4)
	funcName := runtime.FuncForPC(pc).Name()

	//获取上一层的stack
	index := lastIndexByte(f, os.PathSeparator)
	if index != -1 {
		f = f[index+1:]
	}
	return path.Base(funcName) + " " + f + ":" + strconv.Itoa(line) + " "
}
