package base

import (
	"bytes"
	"io"
	"msgPushSite/lib/kfk"
	"msgPushSite/utils"
	"net/http"
	"strconv"
	"time"

	"msgPushSite/internal/context"
	"msgPushSite/internal/glog"
	libip "msgPushSite/lib/ip"
	"msgPushSite/lib/randid"
	"msgPushSite/lib/ws"
	"msgPushSite/mdata"

	"github.com/gin-gonic/gin"
)

func WebRsp(c *context.Context, errCode int, data interface{}, Msg string) {
	if data == nil {
		data = struct{}{}
	}
	respMap := map[string]interface{}{"status_code": errCode, "data": data, "message": Msg}
	c.Header(mdata.HeaderTrace, c.Trace)
	c.JSON(200, respMap)
	return
}

// Handshake websocket连接
func Handshake(c *context.Context) {
	var (
		xApiXXX    = c.Query("xApiXXX")
		clientType = c.Query("clientType")
		siteId     = c.Query("siteId")
	)

	err := ws.Handshake(c, clientType, siteId, xApiXXX, kfk.MsgPushKafka)
	if err != nil {
		glog.Warnf(
			"err=%v |siteId=%s | clientType=%s |xApiXXX=%s ",
			err,
			c.Query("siteId"),
			c.Query("clientType"),
			c.Query("xApiXXX"),
		)
	}
}

func SiteIdMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		c := &context.Context{Context: ctx, Trace: ctx.GetString(mdata.HeaderTrace)}
		// 站点id
		var siteIdHeader = c.GetHeader(mdata.HeaderSite)
		siteId, _ := strconv.Atoi(siteIdHeader)
		if siteId < 1 {
			WebRsp(c, utils.ErrRefuse, nil, utils.MsgIllegalSiteError)
			c.Abort()
			return
		}
		ctx.Request.Header.Set(mdata.HeaderSite, siteIdHeader)
		ctx.Next()
	}
}

func TraceLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先从header获取trace,如果不存在，则直接生成
		var trace = c.GetHeader(mdata.HeaderTrace)
		if trace == "" {
			trace = randid.GenerateId()
		}
		c.Request.Header.Set(mdata.HeaderTrace, trace)

		c.Next()
	}
}

type BodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w BodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w BodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func TraceLoggerMiddlewareDebug() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 过滤掉不需要打印日志的接口
		if mdata.RouterFilterLogPath[c.Request.URL.Path] {
			c.Next()
			return
		}

		bodyLogWriter := &BodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = bodyLogWriter

		//put请求不打印文件内容
		//文件上传只允许用put请求
		params := ""
		if c.Request.Method == http.MethodPost && c.ContentType() != "multipart/form-data" && c.ContentType() != "application/octet-stream" {
			var buf bytes.Buffer
			tee := io.TeeReader(c.Request.Body, &buf)
			requestBody, _ := io.ReadAll(tee)
			c.Request.Body = io.NopCloser(&buf)
			params = string(requestBody)
		}

		cc := context.Background(c)
		timestamp := time.Now()

		c.Next()

		//简单打印日志，调试是否为此处问题
		if _, ok := mdata.SimpleRouterLogPath[c.Request.URL.Path]; ok {
			latency := time.Since(timestamp).Seconds()
			if latency > 20 {
				cc.Tracef(latency, "请求接口%s耗时.2f秒", c.Request.URL.Path, latency)
			}
		}

		var responseBody = "数据过大: "

		//数据过大的内容不放到日志
		if bodyLogWriter.body.Len() < 11<<10 {
			responseBody = bodyLogWriter.body.String()
		} else {
			responseBody += strconv.Itoa(bodyLogWriter.body.Len())
		}

		if c.GetHeader(mdata.HeaderXXX) == "" {
			cc.Tracef(
				time.Since(timestamp).Seconds(),
				"FROM=%s |path=%s |raw=%s |TO=%s |hostName=%s referer=%s |latency=%fs |client_ip=%s |content_length=%s",
				c.Request.URL.Path,
				params,
				c.Request.URL.Path,
				c.Request.URL.RawQuery,
				responseBody,
				c.Request.Host,
				c.Request.Referer(),
				time.Since(timestamp).Seconds(),
				libip.ClientIP(cc),
				c.GetHeader("Content-Length"), //输出content长度，查询慢速查询攻击
			)
		} else {
			cc.Tracef(
				time.Since(timestamp).Seconds(),
				"FROM=%s |path=%s |raw=%s |TO=%s |hostName=%s referer=%s |latency=%fs |client_ip=%s |content_length=%s |xxx=%s",
				c.Request.URL.Path,
				params,
				c.Request.URL.Path,
				c.Request.URL.RawQuery,
				responseBody,
				c.Request.Host,
				c.Request.Referer(),
				time.Since(timestamp).Seconds(),
				libip.ClientIP(cc),
				c.GetHeader("Content-Length"), //输出content长度，查询慢速查询攻击
				c.GetHeader(mdata.HeaderXXX),  //输出解密后的xxx
			)
		}
	}
}
