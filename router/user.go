package router

import (
	"msgPushSite/controller/health"
	"msgPushSite/controller/user/hook"
	"msgPushSite/controller/user/login"
	"msgPushSite/lib/ws"
	"msgPushSite/mdata"
)

// 用户相关websocket路由注册
func userWssRouter() {
	ws.RegisterEndpoint(mdata.MsgIdTypeHeartbeat, health.Health)                      // 健康监测
	ws.RegisterEndpoint(mdata.MsgIDTypeNoLogin, login.NOLogin)                        // 免用户登录授权
	ws.RegisterEndpoint(mdata.MsgIDTypeApiApp, login.ApiApp)                          // WS 推送加密后的 API 下发 APP 的文件内容
	ws.RegisterEndpoint(mdata.MsgIDTypeLogin, login.Login)                            // 登陆授权
	ws.RegisterEndpoint(mdata.MsgIDTypeJoinRoom, login.JoinRoom)                      // 加入聊天室（切换房间）
	ws.RegisterEndpoint(mdata.MsgIDTypeBroadcastRoom, login.BroadcastRoom)            // 普通群聊
	ws.RegisterEndpoint(mdata.MsgIDTypeShareBetRecord, login.BroadcastRoom)           // 分享注单 msgID不同。方法一样
	ws.RegisterEndpoint(mdata.MsgIDTypeChatHistory, login.GetHistoryRecord)           // 获取历史聊天记录 使用http接口
	ws.RegisterEndpoint(mdata.MsgIDTypeLeaveRoom, login.LeaveRoom)                    // 离开房间
	ws.RegisterEndpoint(mdata.MsgIdRedPackageRain, login.BroadRedPackageRain)         // 红包雨消息处理
	ws.RegisterEndpoint(mdata.MsgIdRedPackageReceive, login.ReceiveRedPackageMessage) // 获取到红包雨的消息
}

// 添加hook
func initHook() {
	ws.SetConnectionCreate(hook.OnConnectionCreate)
	ws.SetConnectionStop(hook.OnConnectionStop)
}
