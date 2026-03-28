package login

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"msgPushSite/config"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/db/sqldb"
	"msgPushSite/internal/glog"
	"msgPushSite/mdata/rediskey"
	"strconv"
	"strings"
	"time"

	"msgPushSite/controller/user/base"
	"msgPushSite/lib/ws"
	app "msgPushSite/lib/ws"
	"msgPushSite/mdata"
	"msgPushSite/service/memer"
	"msgPushSite/utils"
)

// NOLogin 免用户登陆接口 建立连接后发送通用配置 不用检验token 用户信息
func NOLogin(c *ws.Context, packet *ws.Packet, rsp *ws.Payload, msg *ws.Msg) (msgFlag ws.MsgFlag) {
	// 1. 检测
	res, err := memer.NOLoginService(c)
	if err != nil {

		// TODO 错误信息先注释掉
		//c.Error(err)
		base.Resp(err, rsp)
		msgFlag = ws.MsgFlagGlobal
		return
	}
	rsp.ResponseSetData(res)
	msgFlag = ws.MsgFlagSelf
	return
}

// WS 推送加密后的 API 下发 APP 的文件内容
func ApiApp(c *ws.Context, packet *ws.Packet, rsp *ws.Payload, msg *ws.Msg) (msgFlag ws.MsgFlag) {
	// 1. 检测
	res, err := memer.NOLoginWSApiApp(c)
	if err != nil {
		c.Error(err)
		base.Resp(err, rsp)
		msgFlag = ws.MsgFlagGlobal
		return
	}
	rsp.ResponseSetData(res)
	msgFlag = ws.MsgFlagSelf
	return
}

// Login 登陆接口
// 广播： 加入房间相关通知
// 单播： VIP发言等级、房间状态、以及房间历史聊天记录等
func Login(c *ws.Context, packet *ws.Packet, rsp *ws.Payload, msg *ws.Msg) (msgFlag ws.MsgFlag) {
	if info, err := c.Member(); err == nil && info != nil {
		glog.Infof("[登录逻辑] :%v", mdata.RepeatLoginErr.Error())
		base.Resp(mdata.RepeatLoginErr, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}

	var req mdata.LoginReqSchema
	err := packet.Decode(&req)
	if err != nil {
		c.Error(err)
		base.Resp(mdata.ArgsParserErr, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}
	// 1. 解析token
	info, err := mdata.ParserToken(&req)
	if err != nil {
		c.Error(err)
		base.Resp(err, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}
	// 2. Client信息 nickname生成/会员信息加载/设置为登陆态
	info.NickName = utils.GenerateNickname(info.Name)
	c.SyncMember(info)
	c.SetLogin()

	//TODO 误信息注释掉
	//glog.Infof(" 登录相关日志: %v 平台:%v", info.Name, c.Client.ClientType())

	// 3. 检测
	res, err := memer.LoginService(c, info, req.RID)
	if err != nil {

		// TODO 错误信息注释掉
		//c.Error(err)
		base.Resp(err, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}
	rsp.ResponseSetData(res)
	msgFlag = ws.MsgFlagSelf
	return
}

// JoinRoom
// 广播： 加入房间信息
// 单播： 房间配置，历史聊天记录
func JoinRoom(c *ws.Context, packet *ws.Packet, rsp *ws.Payload, msgPtr *ws.Msg) (msgFlag ws.MsgFlag) {

	var (
		req mdata.JoinRoomReqSchema
	)
	err := packet.Decode(&req)
	if err != nil {
		c.Error(err)
		base.Resp(mdata.ArgsParserErr, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}
	msgPtr.RoomId = req.Rid
	if room := c.GetRoom(); room != nil {
		c.Infof("当前客户端：%s，所在房间：%s", c.Client.Id, c.Client.GetRoom().GetID())
		if room.GetID() == req.Rid {
			base.Resp(mdata.RepeatJoinRoomErr, rsp)
			msgFlag = ws.MsgFlagSelf
			return
		}
		// 下线切换房间频率
		lock, _ := memer.SetLock(fmt.Sprintf(rediskey.ChangeRoomLimit, c.Id), 1*time.Second)
		if !lock {
			base.Resp(mdata.FrequentOperationLimitErr, rsp)
			msgFlag = ws.MsgFlagSelf
			return
		}
	}
	// 1. 添加这个连接到房间
	info, err := c.Member()
	if err != nil {
		// 2. 如果未登陆，则返回历史聊天记录以及房间信息
		res, err := memer.UnAuthJoinRoomService(c, req.Rid)
		if err != nil {
			c.Error(err)
			base.Resp(mdata.UserNotLoginErr, rsp)
			msgFlag = ws.MsgFlagSelf
			return
		}
		rsp.ResponseSetData(res)
		msgFlag = ws.MsgFlagSelf
		return
	}
	// 3. 获取已登陆数据，聊天记录，房间配置等进行单播  加入房间消息进行广播
	self, err := memer.AuthJoinRoomService(c, req.Rid, info)
	if err != nil {
		base.Resp(err, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}

	// 5. 单播聊天记录以及房间信息
	rsp.ResponseSetData(self)
	msgFlag = ws.MsgFlagSelf

	return
}

// BroadcastRoom 聊天室聊天
func BroadcastRoom(c *ws.Context, packet *ws.Packet, rsp *ws.Payload, msg *ws.Msg) (msgFlag ws.MsgFlag) {
	var (
		req mdata.BroadcastRoomReqSchema
	)
	err := packet.Decode(&req)
	if err != nil {
		c.Error(err)
		base.Resp(mdata.ArgsParserErr, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}
	info, err := c.Member()
	if err != nil || info.Name == "" {
		c.Error(err)
		base.Resp(mdata.UserNotLoginErr, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}
	if len(req.Msg) == 0 {
		base.Resp(mdata.MsgBodyEmptyErr, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}
	res, err := memer.BroadcastVerifyService(c, info, &req, msg)
	if err != nil {
		base.Resp(err, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}

	msgFlag = ws.MsgFlagRoom

	// 标识该条消息是否可举报和是否被当前客户端用户举报过
	/**
	* 这里是新发的消息， 所以肯定是可以举报，并且是没被举报过的， 所以，根据业务要求，这里只有是非单播并且是普通的文本消息，就可以举报； 是否被举报的标识不用动，默认都是否;
		自己发的消息还会推给自己， 这时这个allowreport标识是不对的， 需要前台对比当前消息的发送人和客户端用户名，不一致的才让举报
	*/
	c.Infof("msg.MsgId=>%v|msg.MsgFlag=>%v", msg.MsgId, msgFlag)
	if res.Category == 1 && msgFlag != ws.MsgFlagSelf {
		res.AllowReport = 1
	}
	go func() {
		memer.CacheAndStat(msg, res)
	}()
	rsp.ResponseSetData(res)
	return
}

func GetHistoryRecord(c *ws.Context, packet *ws.Packet, rsp *ws.Payload, msg *ws.Msg) (msgFlag ws.MsgFlag) {
	var (
		req mdata.HistoryRecordReqSchema
	)
	err := packet.Decode(&req)
	if err != nil {
		c.Error(err)
		base.Resp(mdata.ArgsParserErr, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}
	if room := c.GetRoom(); room == nil {
		base.Resp(mdata.NotJoinRoomErr, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}
	lock, err := memer.SetLock(fmt.Sprintf(rediskey.GetHistoryRecordLimit, c.GetSiteId(), c.Id), 1*time.Second)
	if !lock {
		base.Resp(mdata.FrequentOperationLimitErr, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}
	res, total, err := memer.GetHistoryRecord(c, &req)
	if err != nil {
		c.Error(err)
		base.Resp(err, rsp)
		msgFlag = ws.MsgFlagSelf
		return
	}

	//处理消息是否可举报和是否被当前客户端用户举报过的标识
	if len(res) > 0 {
		for k, _ := range res {
			//查询消息是否被自己举报过
			username := c.GetUsername()
			if len(res[k].Seq) > 0 && len(username) > 0 {
				reported, err := sqldb.CheckMsgReportedOrNot(c.GetSiteId(), res[k].Seq, username)
				if err != nil {
					c.Errorf("ws GetHistoryRecord CheckMsgReportedOrNot err|seq=>%v|username=>%v|err=>%v", res[k].Seq, username, err)
				}

				if err == nil {
					if reported {
						//被自己举报过
						res[k].IsReported = 1
					} else {
						//没被自己举报过， 允许举报
						res[k].AllowReport = 1
					}
				}
			}
		}
	}

	p := mdata.PageResp{}
	list := p.Paginator(res, req.PageNum, req.PageSize, total)
	list.CategoryType = req.CategoryType
	list.ChatCategory = req.ChatCategory
	rsp.ResponseOK(list)
	msgFlag = ws.MsgFlagSelf
	return
}

func LeaveRoom(c *ws.Context, packet *ws.Packet, rsp *ws.Payload, msg *ws.Msg) (msgFlag ws.MsgFlag) {
	// 1. 获取房间信息
	room := c.GetRoom()

	//// 2. 如果当前会员已经离开房间，需要删除缓存
	//info, err := c.Member()
	//if err == nil && info.Name != "" && room != nil {
	//_ = memer.DelLevelClients(c.GetSiteId(),info.Name, c.Id, room.GetID())
	//}
	// 3. 解除绑定关系
	if room != nil {
		room.Remove(c.Id)
	}
	// 4. 将当前连接绑定的房间置空
	c.SetRoom(nil)

	rsp.ResponseOK(nil)
	msgFlag = ws.MsgFlagSelf
	return
}

func BroadRedPackageRain(c *ws.Context, packet *ws.Packet, rsp *ws.Payload, msg *ws.Msg) (msgFlag ws.MsgFlag) {
	//ws.SendMsgChan(packet)
	// 通过出消息通道来处理消息

	jsonStr, err := json.Marshal(string(packet.Bytes())) // 会得到 "\"hello world\""
	if err != nil {
		glog.Infof("BroadRedPackageRain err=>%v", err)
	}

	glog.Infof("BroadRedPackageRain jsonStr=>%v", string(jsonStr))

	app.GetApp().RoomPushMethod(utils.Clone(packet.Bytes()), config.GetKafkaTopic().ChatMsgWriteTopic)
	return ws.MsgFlagGlobal

}

func containsVipLevel(vipLevel string, grade string) bool {
	levels := strings.Split(vipLevel, ",") // 按逗号分隔
	for _, v := range levels {
		if strings.TrimSpace(v) == grade {
			return true
		}
	}
	return false
}

func ReceiveRedPackageMessage(c *ws.Context, packet *ws.Packet, rsp *ws.Payload, msg *ws.Msg) (msgFlag ws.MsgFlag) {
	const (
		ActivityTypeGlobal = 0 // 全站
		ActivityTypeSite   = 1 // 默认/站内
	)

	// ===== helpers =====
	respondSelf := func(data any) ws.MsgFlag {
		msg.SiteId = c.SiteId
		msg.MsgData = data
		msg.MsgFlag = ws.MsgFlagSelf
		packet.Reset()
		packet.Packet(msg)
		ws.SendMsgChan(packet)
		return ws.MsgFlagSelf
	}

	// siteId
	siteIDStr := c.GetSiteId()
	siteId, err := strconv.Atoi(siteIDStr)
	if err != nil {
		c.Errorf("invalid siteId: %q, err: %v", siteIDStr, err)
		return respondSelf(nil)
	}

	// Redis 短超时
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// ===== 1) 读取当前活动快照（直接 GetKey 判空，省一次 KeyExist） =====
	currKey := fmt.Sprintf(rediskey.CurrentRedEnvelopeActivityKey, siteId)
	jsonStr, err := core.GetKey(true, currKey)
	if err != nil {
		c.Errorf("GetKey error for %s: %v", currKey, err)
		return respondSelf(nil)
	}
	if jsonStr == "" {
		// 无活动 或 值为空
		return respondSelf(nil)
	}

	result := &mdata.ResEnvelopeMsgVo{}
	if err := mdata.Cjson.Unmarshal([]byte(jsonStr), result); err != nil {
		c.Errorf("unmarshal failed for key=%s, err=%v", currKey, err)
		return respondSelf(nil)
	}

	// ===== 2) 加载 envelope 并做资格过滤（不使用 lazyMember） =====
	var envelopeType int = -1
	var vipLevelStr string
	var activityId int64

	if rdh := rediskey.NewRedEnvelopeHashRedis(result.RedPackId, siteId); rdh != nil {
		if envelope := rdh.GetActivityEnvelope(ctx); envelope != nil {
			envelopeType = envelope.Type
			vipLevelStr = envelope.VipLevel
			activityId = envelope.ActivityId
		}
	}

	switch envelopeType {
	case 1:
		// VIP 过滤
		info, merr := c.Member()
		if merr != nil {
			c.Errorf("Member() error, siteId=%d: %v", siteId, merr)
			return respondSelf(nil)
		}
		if !containsVipLevel(vipLevelStr, strconv.Itoa(info.Vip)) {
			return respondSelf(nil)
		}
	case 0:
		// 指定用户白名单
		info, merr := c.Member()
		if merr != nil {
			c.Errorf("Member() error, siteId=%d: %v", siteId, merr)
			return respondSelf(nil)
		}
		userListKey := fmt.Sprintf(rediskey.CurrentRedEnvelopeUserListKey, siteId, result.RedPackId)
		inList, serr := core.SIsMember(userListKey, info.Name)
		if serr != nil {
			// 出错——更稳妥的做法是不给推送，避免误发
			log.Printf("SIsMember failed, site_id:%d member:%s redPackId:%d activityId:%d err:%v",
				siteId, info.Name, result.RedPackId, activityId, serr)
			return respondSelf(nil)
		}
		if !inList {
			return respondSelf(nil)
		}
	default:
		// 未配置或未知类型：按“全部推送，由客户端判定”策略放行
	}

	// ===== 3) 活动类型：先取值，省一次 KeyExist =====
	result.ActivityType = ActivityTypeSite
	sessionKey := fmt.Sprintf(rediskey.HasRedEnvelopeSessionKey, siteId)
	if v, err := core.GetKey(true, sessionKey); err == nil && v == "1" {
		result.ActivityType = ActivityTypeGlobal
	}

	// ===== 4) 时间戳 & 回包 =====
	now := BJNowTime()
	result.CurrentTime = TimeToMill(now)
	return respondSelf(result)
}

// TimeToMill / 增加一个公共的时间转时间戳
func TimeToMill(t time.Time) int64 {
	return t.UnixNano() / 1e6
}

// BJNowTime 北京当前时间
func BJNowTime() time.Time {
	// 获取北京时间, 在 windows系统上 time.LoadLocation 会加载失败, 最好的办法是用 time.FixedZone, libEs 中的时间为: "2019-03-01T21:33:18+08:00"
	var beiJinLocation *time.Location
	var err error

	beiJinLocation, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		beiJinLocation = time.FixedZone("CST", 8*3600)
	}

	nowTime := time.Now().In(beiJinLocation)

	return nowTime
}
