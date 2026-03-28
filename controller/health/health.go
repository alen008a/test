package health

import (
	"msgPushSite/controller/base"
	"msgPushSite/internal/context"
	"msgPushSite/lib/ws"
	"msgPushSite/utils"
)

func Health(c *ws.Context, packet *ws.Packet, rsp *ws.Payload, msg *ws.Msg) (msgFlag ws.MsgFlag) {
	rsp.StatusCode = utils.StatusOK
	rsp.Message = utils.MsgSuccess
	return
}

func HealthV1(c *context.Context) {
	base.WebRsp(c, 200, nil, utils.MsgSuccess)
}
