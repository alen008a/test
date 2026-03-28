package ws

import (
	"fmt"
	"msgPushSite/internal/glog"
	"runtime"
)

type Middleware func(endpoint Endpoint) Endpoint

func MiddleApply(endpoint Endpoint) Endpoint {
	return wrapper(endpoint)
}

func wrapper(e Endpoint) Endpoint {
	return func(c *Context, req *Packet, rsp *Payload, msg *Msg) (messageFlag MsgFlag) {
		defer func() {
			if err := recover(); err != nil {
				var buf [4096]byte
				n := runtime.Stack(buf[:], false)
				tmpStr := fmt.Sprintf("err=%v panic ==> %s\n", err, string(buf[:n]))
				glog.Emergency(tmpStr)
			}
		}()

		//now := time.Now()
		messageFlag = e(c, req, rsp, msg)
		//c.Infof(
		//	"FROM | Message[%d] |req=%s |TO=%s |latency=%.2f |name=%s |clientID=%s |clientType=%s",
		//	msg.MsgId,
		//	req.String(),
		//	mdata.MustMarshal2String(rsp),
		//	time.Since(now).Seconds(),
		//	c.key,
		//	c.Id,
		//	c.clientType,
		//)
		return
	}
}
