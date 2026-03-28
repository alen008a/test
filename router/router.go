package router

import (
	"msgPushSite/controller/base"
	"msgPushSite/controller/health"
	"msgPushSite/controller/user/api"
	"msgPushSite/internal/x"

	"msgPushSite/config"

	"github.com/gin-gonic/gin"
)

const (
	HttpVOne = "/stream/api/v1"
	WsVOne   = "/stream/ws/v1"
)

// ApiRouter 路由
func ApiRouter(httpRouter *gin.Engine) {

	httpRouter.Use(base.TraceLoggerMiddleware())
	if config.GetApplication().OpenMiddlewareTraceDebug {
		httpRouter.Use(base.TraceLoggerMiddlewareDebug())
	}

	httpV1 := httpRouter.Group(HttpVOne)
	{
		httpV1.Use(base.SiteIdMiddleware())
		x.GET(httpV1, "/health", health.HealthV1)
		x.GET(httpV1, "/history", api.GetHistoryRecord)
		// 测试接口
		x.GET(httpV1, "/create/room", api.CreateRoom)
		x.GET(httpV1, "/create/vip", api.CreateVIPConfig)
		x.GET(httpV1, "/generate/record", api.GenerateBroadcastRecord)
		x.GET(httpV1, "/generate/token", api.GenerateToken)
		x.GET(httpV1, "/generate/batchToken", api.BatchGenerateJWTToken)
		x.POST(httpV1, "/pulse", api.ServePulse)
		x.POST(httpV1, "/history", api.GetHistoryRecord)
		x.POST(httpV1, "/report", api.Report)                     //聊天室消息举报
		x.POST(httpV1, "/getReportReasons", api.GetReportReasons) //聊天室消息举报- 获取举报原因
		//x.POST(httpV1, "/redPackageRain", api.RedPackageRainHandler)
		// 不需要的接口
		//x.POST(httpV1, "/online", api.GetOnlineUser)
		//x.POST(httpV1, "/validToken", api.ValidToken)
		//x.POST(httpV1, "/getDeviceToken", api.GetDeviceToken)
		//x.POST(httpV1, "/msg", api.SendMsg)
	}

	wsV1 := httpRouter.Group(WsVOne)
	{
		x.GET(wsV1, "/handshake", base.Handshake)
	}

	// 用户相关websocket路由
	userWssRouter()
	// 刚创建连接
	initHook()
}
