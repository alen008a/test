package ws

import (
	"path"
	"runtime"
	"strings"

	. "msgPushSite/internal/glog/log"

	"github.com/rs/xid"
)

// Context 使用专用的context，防止其他开发者直接用gin.Conext
type Context struct {
	Trace  string
	SiteId string
	*Client
}

func NewWsContext() *Context {
	c := new(Context)
	c.Trace = xid.New().String()
	return c
}

func (c *Context) Info(args ...interface{}) {
	ZapLog.Named(funcName(c.Trace, c.SiteId)).Info(args...)
}

func (c *Context) Infof(template string, args ...interface{}) {
	ZapLog.Named(funcName(c.Trace, c.SiteId)).Infof(template, args...)
}

func (c *Context) Warn(args ...interface{}) {
	ZapLog.Named(funcName(c.Trace, c.SiteId)).Warn(args...)
}

func (c *Context) Warnf(template string, args ...interface{}) {
	ZapLog.Named(funcName(c.Trace, c.SiteId)).Warnf(template, args...)
}

func (c *Context) Error(args ...interface{}) {
	ZapLog.Named(funcName(c.Trace, c.SiteId)).Error(args...)
}

func (c *Context) Errorf(template string, args ...interface{}) {
	ZapLog.Named(funcName(c.Trace, c.SiteId)).Errorf(template, args...)
}

func funcName(args ...string) string {
	pc, _, _, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()
	return path.Base(funcName) + " " + strings.Join(args, " ")
}
