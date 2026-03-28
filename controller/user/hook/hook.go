package hook

import (
	"msgPushSite/internal/glog"
	"msgPushSite/lib/ws"
	"msgPushSite/service/memer"
	"msgPushSite/utils"
	"time"
)

func OnConnectionCreate(client *ws.Client) {

	//TODO 临时注释
	//glog.Infof("Remote address create connection: [%s] |  SiteId :[%s] | Connection ID: [%s] | ClientType: [%s] | GetUsername: [%s] |  Time:[%s]",
	//	client.Ip(),
	//	client.GetSiteId(),
	//	client.Id,
	//	client.ClientType(),
	//	client.GetUsername(),
	//	time.Now().Format(utils.TimeBarFormat),
	//)
	// TODO 总计数，IP连接数限制，IP黑名单等
}

func OnConnectionStop(client *ws.Client) {
	glog.Infof("Stop remote connection: [%s] |  SiteId :[%s]  | Connection ID: [%s] | ClientType: [%s] | GetUsername: [%s] | Time:[%s]",
		client.Ip(),
		client.GetSiteId(),
		client.Id,
		client.ClientType(),
		client.GetUsername(),
		time.Now().Format(utils.TimeBarFormat),
	)
	info, err := client.Member()
	if err != nil || info.Name == "" {
		return
	}
	_ = memer.DelCloseClient(client, info)
	// TODO 如果区分客户端连接，当会员断线，需要更新会员

}
