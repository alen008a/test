package memer

import (
	"errors"
	"fmt"
	"msgPushSite/config"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/internal/glog"
	"msgPushSite/lib/es"
	"msgPushSite/lib/kfk"
	"msgPushSite/lib/ws"
	"msgPushSite/mdata"
	"msgPushSite/mdata/rediskey"
	"strings"
	"time"
)

// NOLoginService 免检验token 下发通知给客户端
func NOLoginService(c *ws.Context) (*mdata.LoginRspSchema, error) {
	var (
		err error
		res = new(mdata.LoginRspSchema)
	)

	// 1. 获取当前会员达到发言VIP等级
	vip, err := getVIPConfig(c)
	if err != nil {
		return nil, err
	}
	res.VipMin = vip.VipMin
	res.VipMax = vip.VipMax
	res.EffectMin = vip.EffectMin
	res.EffectMax = vip.EffectMax
	res.EffectOpen = vip.EffectOpen
	res.BsOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.BetsStrategiesOpen, c.GetSiteId()))
	res.AdvOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.AdvStrategiesOpen, c.GetSiteId()))
	res.GiftOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.LiveGiftStatusOpen, c.GetSiteId()))
	res.CopyOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.ChatIsCanCopyContentOpen, c.GetSiteId()))
	res.ScoreOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.ChatMatchScoreStatusOpen, c.GetSiteId()))
	res.IpServer = IpServerAddress(c.GetSiteId(), c.ClientType())
	// 2. 如果加入房间，登陆后需要在房间内广播
	if room := c.GetRoom(); room != nil {
		err := PromptJoinRoom(c, vip)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// NOLoginWSApiApp 免检验token 下发通知给客户端 WS 推送加密后的 API 下发 APP 的文件内容
func NOLoginWSApiApp(c *ws.Context) (*mdata.LoginRspSchema, error) {
	var (
		res = new(mdata.LoginRspSchema)
	)
	res.BsOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.BetsStrategiesOpen, c.GetSiteId()))
	res.AdvOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.AdvStrategiesOpen, c.GetSiteId()))
	res.GiftOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.LiveGiftStatusOpen, c.GetSiteId()))
	res.CopyOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.ChatIsCanCopyContentOpen, c.GetSiteId()))
	res.ScoreOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.ChatMatchScoreStatusOpen, c.GetSiteId()))
	res.IpServerApp = WSApiApp(c.SiteId, c.ClientType())
	return res, nil
}

// LoginService 校验token-》查询当前用户状态-》获取房间状态/VIP发言等级-》下发给客户端
func LoginService(c *ws.Context, info *mdata.MemberInfo, rid string) (*mdata.LoginRspSchema, error) {
	var (
		err error
		res = new(mdata.LoginRspSchema)
	)
	// 1. 检测当前会员是否被禁言
	status := GetIsBan(info.SiteId, info.Name)
	res.SpeechStatus = 0
	if status {
		res.SpeechStatus = 1
	}
	notify, speechStatus := MemberBannedStatus(info.SiteId, info.Name, res.SpeechStatus)
	// 2. 获取当前会员达到发言VIP等级
	vip, err := getVIPConfig(c)
	if err != nil {
		return nil, err
	}
	res.SpeechStatus = speechStatus
	res.Notify = notify
	res.VipMin = vip.VipMin
	res.VipMax = vip.VipMax
	res.EffectMin = vip.EffectMin
	res.EffectMax = vip.EffectMax
	res.EffectOpen = vip.EffectOpen
	res.BsOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.BetsStrategiesOpen, info.SiteId))
	res.AdvOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.AdvStrategiesOpen, info.SiteId))
	res.GiftOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.LiveGiftStatusOpen, info.SiteId))
	res.CopyOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.ChatIsCanCopyContentOpen, info.SiteId))
	res.ScoreOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.ChatMatchScoreStatusOpen, info.SiteId))

	// 4. 将会员信息写入redis 这里因不使用 故注释
	// data, err := mdata.Cjson.Marshal(info)
	// if err != nil {
	// 	c.Errorf("LoginService mdata.Cjson.Marshal is error: %s", err.Error())
	// 	return nil, mdata.SerViceStatusErr
	// }
	//err = core.SetExpireKV(fmt.Sprintf(rediskey.SiteActiveMemberInfo, info.Name), string(data), 24*time.Hour)
	// if err != nil {
	// 	c.Errorf("LoginService SetNotExpireKV is error: %s", err.Error())
	// 	return nil, mdata.SerViceStatusErr
	// }
	// // 5. 将会员用户名与连接ID进行绑定, 将该连接设置为登录态
	//err = core.SetExpireKV(fmt.Sprintf(rediskey.ClientMemberName, info.Name), c.Id, 24*time.Hour)
	// if err != nil {
	// 	c.Errorf("LoginService SetNotExpireKV %s is error: %s", fmt.Sprintf("CLIENT_MEMBER_NAME_%s", info.Name), err.Error())
	// 	return nil, mdata.SerViceStatusErr
	// }
	// 6. 当会员已经加入房间时，需要走以下逻辑
	//err = loginUpdateRoom(c, info)
	//if err != nil {
	//	return nil, err
	//}

	// 7. 如果加入房间，登陆后需要在房间内广播
	if room := c.GetRoom(); room != nil {
		PromptJoinRoom(c, vip)
	}

	// 8。用户在线状态同步redis
	c.Client.GetHub().SaveMemberToRedis(c.Client)
	// 8. 内部通知消息队列补发消息
	go asynPushHistoryNotice(c, info.Name)

	return res, nil
}

// PromptJoinRoom 推送加入房间欢迎语
func PromptJoinRoom(c *ws.Context, vip *mdata.VIPConf) (err error) {

	if c.GetRoom() == nil {
		return errors.New("invalid room")
	}

	rid := c.GetRoom().GetID()

	//如果当前链接加入过该房间 不需要广播
	if _, ok := c.GetProperty(rid); ok {
		return
	}

	// 1. 如果没有登陆 则跳过
	info, err := c.Member()
	if err != nil || info == nil {
		return
	}
	// 2. 获取VIP信息
	if vip == nil {
		vip, err = getVIPConfig(c)
		if err != nil {
			return
		}
	}
	pushJoinEffect(c, vip, info, rid, mdata.MsgIDTypeBroadcastJoin)
	return
}

func pushJoinEffect(c *ws.Context, vip *mdata.VIPConf, info *mdata.MemberInfo, rid string, msgType uint32) {
	channelType := ws.MsgFlagSelf
	broadcast := new(mdata.AuthJoinRoomBroadcastRspSchema)
	broadcast.Msg = "进入聊天室"
	broadcast.VIP = info.Vip
	broadcast.Nickname = info.NickName
	broadcast.MemberId = info.Id
	broadcast.Category = 3 // 表示加入房间类型消息
	if vip.EffectMin <= info.Vip && info.Vip <= vip.EffectMax && vip.EffectOpen == 1 {
		broadcast.EffectStatus = 1
		channelType = ws.MsgFlagRoom
	}
	c.SetProperty(rid, rid)
	p := new(ws.Payload)
	p.ResponseOK(broadcast)
	msg := ws.NewMsg(channelType, msgType)
	msg.RoomId = c.GetRoom().GetID()
	msg.MsgData = p
	msg.SiteId = info.SiteId
	msg.EsIndexName = fmt.Sprintf(es.ESIndexPrefix, info.SiteId) + time.Now().Format("2006_01")
	pack := ws.NewPacket()
	pack.Reset()
	pack.Packet(msg)
	kfk.MsgPushKafka(pack.Bytes(), config.GetKafkaTopic().ChatMsgWriteTopic)
	pack.Release()
	//记录房间在线人数和进线人数
	activeKey := fmt.Sprintf(rediskey.LiveRoomActiveTotal, rid)
	core.SAdd(activeKey, info.Name)
	addRoomTTl(activeKey, 5)
	enterKey := fmt.Sprintf(rediskey.LiveRoomEnterTotal, rid)
	core.SAdd(enterKey, info.Name)
	addRoomTTl(enterKey, 5)
}

func asynPushHistoryNotice(c *ws.Context, name string) {
	key := fmt.Sprintf(rediskey.ClientNoticeSet, c.GetSiteId(), name, strings.ToLower(c.ClientType()))
	now := time.Now()
	start := now.Add(-3 * 24 * time.Hour)
	vals1, _ := core.ZRangeByScore(true, key, fmt.Sprintf("%d", start.Unix()), fmt.Sprintf("%d", now.Unix()))
	glog.Debugf("异步历史定向消息 name: %s, vals1: %s", name, vals1)

	key = fmt.Sprintf(rediskey.ClientNoticeSet, c.GetSiteId(), "*", strings.ToLower(c.ClientType()))
	vals2, _ := core.ZRangeByScore(true, key, fmt.Sprintf("%d", start.Unix()), fmt.Sprintf("%d", now.Unix()))
	glog.Debugf("异步历史广播消息 name: %s, vals2: %s", "*", vals2)

	vals := append(vals1, vals2...)
	var hmsgs []*ws.HistoryMsg
	for _, v := range vals {
		s, _ := core.GetKey(false, v)
		hmsg := &ws.HistoryMsg{}
		mdata.Cjson.UnmarshalFromString(s, &hmsg)
		glog.Debugf("异步历史广播消息 name: %s key: %s, member: %s, hmsg: %s", "*", key, v, s)
		if hmsg.MsgId != mdata.MsgIDInternalNotice {
			glog.Debugf("异步历史广播消息 name: %s, msg.MsgId 错误, msg: %s", "*", v)
			continue
		}

		if hmsg.MsgFlag == ws.MsgFlagConditionGlobal { //全局条件广播
			k := fmt.Sprintf(rediskey.ClientNoticeRecordHash, c.GetSiteId(), hmsg.Seq, strings.ToLower(c.ClientType()))
			v, _ := core.HGet(false, k, name)
			if v == "1" { //1: 已经发送过了
				continue
			}
		}

		hmsgs = append(hmsgs, hmsg)
	}

	if len(hmsgs) > 10 {
		hmsgs = hmsgs[:10]
	}

	for _, hmsg := range hmsgs {
		msg := &ws.Msg{
			SiteId:      c.GetSiteId(),
			Seq:         hmsg.Seq,
			MsgFlag:     hmsg.MsgFlag,
			Key:         hmsg.Key,
			MsgId:       hmsg.MsgId,
			MsgData:     hmsg.MsgData,
			ClientTypes: hmsg.ClientTypes,
			IsAgent:     hmsg.IsAgent,
			Trace:       hmsg.Trace,
			EsIndexName: fmt.Sprintf(es.ESIndexPrefix, c.GetSiteId()) + time.Now().Format("2006_01"),
		}

		pack := ws.NewPacket()
		pack.Reset()
		pack.Packet(msg)
		kfk.MsgPushKafka(pack.Bytes(), config.GetKafkaTopic().ChatMsgWriteTopic)
		pack.Release()
		time.Sleep(3 * time.Second)
	}
}
