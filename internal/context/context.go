package context

import (
	"fmt"
	"msgPushSite/config"
	"msgPushSite/internal/glog/bot"
	. "msgPushSite/internal/glog/log"
	"msgPushSite/mdata"
	"path"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
)

type Handler func(ctx *Context)

type Context struct {
	*gin.Context
	SiteId string
	Trace  string
}

func Background(c *gin.Context) *Context {
	return &Context{c, c.GetHeader(mdata.HeaderTrace), c.GetHeader(mdata.HeaderSite)}
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

func (c *Context) GetClientIP() string {
	clientIP := c.Request.Header.Get("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(c.Request.Header.Get("X-Real-Ip"))
	}
	if clientIP != "" {
		return clientIP
	}
	return c.ClientIP()
}

func funcName(args ...string) string {
	pc, _, _, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()
	return path.Base(funcName) + " " + strings.Join(args, " ")
}

// 仅提供给中间件使用，告警慢查询接口
func (c *Context) Tracef(latency float64, template, path string, args ...interface{}) {
	if latency > 20 {
		bot.SendMid(c.SiteId, fmt.Sprintf(bot.SlowTemplate, config.GetApplication().AppID, c.SiteId+config.GetApplication().Cluster, path, latency, c.Trace), path)
	}
	ZapLog.Named(funcName(c.Trace)).Infof(template, args...)
}
