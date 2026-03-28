package api

import (
	ctx "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"msgPushSite/config"
	"msgPushSite/controller/base"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/db/sqldb"
	"msgPushSite/internal/context"
	"msgPushSite/lib/es"
	"msgPushSite/lib/httpclient"
	"msgPushSite/lib/kfk"
	"msgPushSite/lib/randid"
	"msgPushSite/lib/ws"
	app "msgPushSite/lib/ws"
	"msgPushSite/mdata"
	"msgPushSite/mdata/rediskey"
	"msgPushSite/service/memer"
	"msgPushSite/service/metadata"
	"msgPushSite/utils"
	"os"
	"strconv"
	"time"

	redis "msgPushSite/db/redisdb/core"
)

type Room struct {
	RoomID   string `json:"roomId"`                           // 房间ID
	Status   int    `gorm:"column:status" json:"status"`      // 状态 启用1 停用0
	LiveDate string `gorm:"column:live_date" json:"liveDate"` // 开赛时间
}

func CreateRoom(c *context.Context) {
	var (
		err error
		req = new(Room)
	)
	err = c.ShouldBindJSON(req)
	if err != nil {
		base.WebRsp(c, utils.ErrInvalidParams, nil, utils.MsgInvalidParamsError)
		return
	}
	roomByte, err := mdata.Cjson.Marshal(req)
	if err != nil {
		base.WebRsp(c, utils.ErrInternal, nil, utils.MsgNotFoundError)
		return
	}
	err = core.SetNotExpireKV(fmt.Sprintf(rediskey.LiveMatch, metadata.GetSiteIdString(c), req.RoomID), string(roomByte))
	if err != nil {
		base.WebRsp(c, utils.ErrInternal, nil, utils.MsgSuccess)
		return
	}
	base.WebRsp(c, utils.StatusOK, nil, fmt.Sprintf(rediskey.LiveMatch, metadata.GetSiteIdString(c), req.RoomID))
}

func CreateVIPConfig(c *context.Context) {
	var (
		err error
		req = new(mdata.VIPConf)
	)
	err = c.ShouldBindJSON(req)
	if err != nil {
		base.WebRsp(c, utils.ErrInvalidParams, nil, utils.MsgInvalidParamsError)
		return
	}
	c.Info("CreateVIPConfig schema: ", *req)
	vipByte, err := mdata.Cjson.Marshal(req)
	if err != nil {
		base.WebRsp(c, utils.ErrInternal, nil, utils.MsgNotFoundError)
		return
	}
	err = core.SetNotExpireKV(fmt.Sprintf(rediskey.ChatVIPConf, metadata.GetSiteIdString(c)), string(vipByte))
	if err != nil {
		base.WebRsp(c, utils.ErrInternal, nil, "设置VIP等级出错")
		return
	}
	base.WebRsp(c, utils.StatusOK, nil, utils.MsgSuccess)
}

type GenerateTokenSchema struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Vip      int    `json:"vip"`
	Token    string `json:"token"`
	CreateAt string `json:"createAt"`
	Count    int    `json:"count"`
}

func GenerateToken(c *context.Context) {
	var (
		err error
		req = new(GenerateTokenSchema)
	)
	err = c.ShouldBindJSON(req)
	if err != nil {
		base.WebRsp(c, utils.ErrInvalidParams, nil, utils.MsgInvalidParamsError)
		return
	}
	token, err := mdata.GenerateToken(req.Id, req.Vip, req.Name, req.Token, req.CreateAt)
	if err != nil {
		base.WebRsp(c, utils.ErrInvalidParams, nil, utils.MsgInvalidParamsError)
		return
	}
	base.WebRsp(c, utils.StatusOK, map[string]string{"jwtToken": token}, utils.MsgSuccess)
}

func BatchGenerateJWTToken(c *context.Context) {
	var (
		err error
		req = new(GenerateTokenSchema)
		rsp = make([]map[string]interface{}, 0)
	)
	err = c.ShouldBindJSON(req)
	if err != nil {
		base.WebRsp(c, utils.ErrInvalidParams, nil, utils.MsgInvalidParamsError)
		return
	}
	if req.Count <= 0 && req.Count >= 100000 {
		base.WebRsp(c, utils.ErrInvalidParams, nil, utils.MsgInvalidParamsError)
		return
	}
	for i := 1; i < req.Count; i++ {
		rand.Seed(time.Now().UnixNano())
		req.Id = rand.Intn(10000000000)
		req.Vip = rand.Intn(10)
		req.Name = RandStringRunes(8)
		req.CreateAt = time.Now().Add(-(time.Hour * 24 * time.Duration(rand.Intn(1000)))).Format(utils.TimeBarFormat)
		token, err := mdata.GenerateToken(req.Id, req.Vip, req.Name, req.Token, req.CreateAt)
		if err != nil {
			base.WebRsp(c, utils.ErrInvalidParams, nil, utils.MsgInvalidParamsError)
			return
		}
		ele := make(map[string]interface{})
		ele["id"] = req.Id
		ele["name"] = req.Name
		ele["vip"] = req.Vip
		ele["jwt"] = token
		rsp = append(rsp, ele)
	}
	base.WebRsp(c, utils.StatusOK, rsp, utils.MsgSuccess)
}

func GenerateBroadcastRecord(c *context.Context) {
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 50; i++ {
		msg := new(ws.Msg)
		msg.Seq = randid.GenerateId()
		msg.MsgFlag = 1
		msg.RoomId = "abel001"
		msg.Key = RandStringRunes(8)
		msg.MsgId = 10003
		msg.MsgData = &ws.Payload{
			StatusCode: 6000,
			Message:    "success",
			Data: &mdata.BroadcastRoomRspSchema{
				Nickname:  utils.GenerateNickname(msg.Key),
				VIP:       10,
				Msg:       RandStringRunes(100),
				Timestamp: time.Now().Format(utils.TimeBarFormat),
				MemberId:  10023,
				Category:  1,
			},
		}
		dataByte, err := mdata.Cjson.Marshal(msg)
		if err != nil {
			base.WebRsp(c, utils.ErrInternal, nil, utils.MsgInternalError)
			return
		}
		kfk.MsgPushKafka(dataByte, config.GetKafkaTopic().ChatMsgWriteTopic)
	}

	base.WebRsp(c, utils.StatusOK, nil, utils.MsgSuccess)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func GetHistoryRecord(c *context.Context) {
	req := new(mdata.HistoryRecordReqHTTPSchema)
	period1 := time.Now()
	err := mdata.Cjson.NewDecoder(c.Request.Body).Decode(req)
	if err != nil {
		if err == io.EOF {
			c.Errorf("历史聊天记录查询，读取用户输入EOF，耗时：%.2f秒", time.Since(period1).Seconds())
			period2 := time.Now()
			base.WebRsp(c, utils.ErrInternal, nil, "读取用户输入失败")
			c.Infof("历史聊天记录查询，输出响应耗时：%.2f秒", time.Since(period2).Seconds())
		} else {
			c.Errorf("GetHistoryRecord error: %v", err)
			base.WebRsp(c, utils.ErrInternal, nil, "读取用户输入失败")
		}
		return
	}
	duration := time.Second * 40
	cancelCtx, cancelFunc := ctx.WithTimeout(c.Context, duration)
	defer cancelFunc()

	// create a done channel to tell the request it's done
	doneChan := make(chan *mdata.ResultPack, 1)
	// here you put the actual work needed for the request
	// and then send the doneChan with the status and body
	// to finish the request by writing the response
	go func(ctx ctx.Context) {
		doneChan <- getHistoryList(c, req)
	}(cancelCtx)
	// non-blocking select on two channels see if the request
	// times out or finishes
	select {
	// if the context is done it timed out or was cancelled
	// so don't return anything
	case <-cancelCtx.Done():
		c.Errorf("读取历史聊天记录超时")
		base.WebRsp(c, utils.ErrInternal, nil, utils.MsgTimeOut)
		return
		// if the request finished then finish the request by
		// writing the response
	case res := <-doneChan:
		base.WebRsp(c, res.ErrCode, res.List, res.ErrorMsg)
	}
}

func getHistoryList(c *context.Context, req *mdata.HistoryRecordReqHTTPSchema) *mdata.ResultPack {
	var (
		err    error
		result = new(mdata.ResultPack)
	)
	// 1. 校验参数
	if req.RID == "" && req.Seq == "" {
		c.Errorf("GetHistoryRecord error params: %+v", req)
		result.ErrCode = utils.ErrInvalidParams
		result.ErrorMsg = utils.MsgInvalidParamsError
		return result
	}
	// 2. 校验token
	loginReq := mdata.LoginReqSchema{
		Token: req.Token,
		Body:  req.Body,
	}
	info, err := mdata.ParserToken(&loginReq)
	if err != nil || info == nil || info.Name == "" {
		if errors.As(err, &mdata.TokenExpireErr) {
			result.ErrCode = utils.ErrTokenExpired
			result.ErrorMsg = err.Error()
			return result
		}
		c.Errorf("GetHistoryRecord error: %v", err)
		result.ErrCode = utils.ErrRefuse
		result.ErrorMsg = "请先进行登录"
		return result
	}
	// 3. 查询历史聊天记录
	historyReq := mdata.HistoryRecordReqSchema{
		SiteId:       info.SiteId,
		RID:          req.RID,
		PageNum:      req.PageNum,
		PageSize:     req.PageSize,
		ChatCategory: req.ChatCategory, //0表所有 1表聊天 2表晒单
		CategoryType: req.CategoryType, //0 所有 如果 category晒单 ，type -1为普通单 2 为大单
		Seq:          req.Seq,
		BeginTime:    req.BeginTime, //查询起始时间
		EndTime:      req.EndTime,   //查询截止时间

	}
	res, total, err := memer.GetHistoryRecords(c, &historyReq)
	if err != nil {
		c.Errorf("GetHistoryRecord error: %v", err)
		result.ErrCode = utils.ErrInternal
		result.ErrorMsg = utils.MsgInternalError
		return result
	}

	//处理消息是否可举报和是否被当前客户端用户举报过的标识
	if len(res) > 0 {
		for k, _ := range res {
			//查询消息是否被自己举报过
			if len(res[k].Seq) > 0 && len(info.Name) > 0 {
				reported, err := sqldb.CheckMsgReportedOrNot(info.SiteId, res[k].Seq, info.Name)
				if err != nil {
					c.Errorf("http GetHistoryRecord CheckMsgReportedOrNot err|seq=>%v|username=>%v|err=>%v", res[k].Seq, info.Name, err)
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
	result.List = list
	result.ErrCode = utils.StatusOK
	result.ErrorMsg = utils.MsgSuccess
	return result
}

func ServePulse(c *context.Context) {

	pulse := ws.Pulse()

	hostname, _ := os.Hostname()
	pulse.Server = hostname

	base.WebRsp(c, utils.StatusOK, pulse, "health")
}

// 聊天室消息举报
func Report(c *context.Context) {

	//获取并校验校验参数
	req := &mdata.LiveMatchMsgReportParams{}
	err := c.ShouldBindJSON(req)
	if err != nil {
		c.Errorf("Report decode params error|params=>%#v|err=>%v", req, err)
		base.WebRsp(c, utils.ErrInvalidParams, nil, utils.MsgInvalidParamsError)
		return
	}

	//校验必要参数
	if len(req.Token) < 1 {
		base.WebRsp(c, utils.ErrInvalidParams, nil, "token不能为空")
		return
	}
	if len(req.Body) < 1 {
		base.WebRsp(c, utils.ErrInvalidParams, nil, "body不能为空")
		return
	}
	if len(req.Seq) < 1 {
		base.WebRsp(c, utils.ErrInvalidParams, nil, "消息唯一序列号不能为空")
		return
	}
	if len(req.Reason) < 1 {
		base.WebRsp(c, utils.ErrInvalidParams, nil, "举报原因不能为空")
		return
	}

	//校验原因是否合法
	if !mdata.CheckMsgReportReason(req.Reason) {
		base.WebRsp(c, utils.ErrInvalidParams, nil, "非法的举报原因")
		return
	}

	//获取登录的会员信息
	memberInfo, err := mdata.ParserToken(&mdata.LoginReqSchema{
		Token: req.Token,
		Body:  req.Body,
	})
	if err != nil {
		c.Errorf("Report ParserToken error|err=>%v", err)
		if errors.Is(err, mdata.TokenExpireErr) {
			base.WebRsp(c, utils.ErrTokenExpired, nil, err.Error())
			return
		}
		base.WebRsp(c, utils.ErrTokenInvalid, nil, "解析会员token异常")
		return
	}
	if len(memberInfo.Name) < 1 {
		base.WebRsp(c, utils.ErrInternal, nil, "会员信息异常")
		return
	}

	//校验该会员是否可以举报(1.每人每天只能举报N次， 次数由后台配置(默认10)   2.一条消息只能被同一个人举报一次)
	//1.查询会员今天举报的次数， 再查询后台配置的数据，进行对比
	memberDailyReportCountRedisKey := fmt.Sprintf(mdata.MemberReportCountRedisKey, metadata.GetSiteIdString(c), memberInfo.Name, time.Now().Format(utils.TimeYYMMDD))
	memberReportCountTodayString, err := core.GetKey(false, memberDailyReportCountRedisKey)
	memberReportCountToday, _ := strconv.Atoi(memberReportCountTodayString)
	if len(memberReportCountTodayString) < 1 {
		c.Errorf("Report redis get memberReportCountToday err|redis key=>%v|result=>%v|err=>%v", memberDailyReportCountRedisKey, memberReportCountTodayString, err)
	}

	maxCount := mdata.DefaultMaxReportCount
	configMaxCountString, err := core.GetKey(false, fmt.Sprintf(mdata.MaxReportCountRedisKey, metadata.GetSiteIdString(c)))
	configMaxCount, _ := strconv.Atoi(configMaxCountString)
	if len(configMaxCountString) < 1 {
		c.Errorf("Report redis get DefaultMaxReportCount err|redis key=>%v|result=>%v|err=>%v", fmt.Sprintf(mdata.MaxReportCountRedisKey, metadata.GetSiteIdString(c)), configMaxCountString, err)
	}
	if err == nil && configMaxCount > 0 {
		maxCount = configMaxCount
	}
	if memberReportCountToday >= maxCount {
		base.WebRsp(c, utils.ErrInternal, nil, "举报次数已达上限，请隔天再试")
		return
	}

	//2.查询会员是否有举报过此条消息， 不能重复举报
	isReported, err := sqldb.CheckMsgReportedOrNot(metadata.GetSiteIdString(c), req.Seq, memberInfo.Name)
	if err != nil {
		c.Errorf("Report CheckMsgReportedOrNot err|seq=>%v|name=>%v|err=>%v", req.Seq, memberInfo.Name, err)
		base.WebRsp(c, utils.ErrInternal, nil, "查询举报记录异常")
		return
	}
	if isReported {
		base.WebRsp(c, utils.ErrInternal, nil, "您已举报过该消息")
		return
	}

	//获取被举报的消息数据
	var msgInfo = new(mdata.LiveMatchMessage)
	var err1 error
	c.Infof("Report es config|true")
	esMsgInfo, err := es.GetMsgDataBySeqFromES(&mdata.HistoryRecordReqSchema{
		SiteId: metadata.GetSiteIdString(c),
		Seq:    req.Seq,
	})
	msgInfo = esMsgInfo
	err1 = err
	if err1 != nil {
		c.Errorf("Report get msg data error|seq=>%v|err=>%v", req.Seq, err1)
		base.WebRsp(c, utils.ErrInternal, nil, "获取消息数据异常-1")
		return
	}

	//赛事id为空的不拦截， 继续往下走， 当入库的举报记录数据中出现字段为空的， 考虑后续用脚本修复
	if len(msgInfo.MatchId) < 1 {
		c.Errorf("Report msgInfo.MatchId empty|msgInfo=>%#v", msgInfo)
	}

	//只允许举报常规的文本消息
	if msgInfo.Category != 1 {
		base.WebRsp(c, utils.ErrInternal, nil, "不可举报该类型的消息")
		return
	}
	//不能举报自已发送的消息
	if msgInfo.Name == memberInfo.Name {
		base.WebRsp(c, utils.ErrInternal, nil, "不支持自己举报自己")
		return
	}

	//获取被举报的消息所属的比赛数据
	matchData, err := sqldb.GetMatchDataByMatchID(metadata.GetSiteIdString(c), msgInfo.MatchId)
	//未成功获取到赛事数据时， 不拦截， 继续往下走， 当入库的举报记录数据中出现字段为空的， 考虑后续用脚本修复
	if err != nil {
		c.Errorf("Report GetMatchDataByMatchID error|matchId=>%v|err=>%v", msgInfo.MatchId, err)
	}

	//构造举报数据，入库
	insertMsgReportData := &mdata.LiveMatchMessageReport{
		SiteId:       metadata.GetSiteId(c),
		Msg:          msgInfo.Msg,
		Seq:          msgInfo.Seq,
		Name:         msgInfo.Name,
		Vip:          msgInfo.Vip,
		ReporterName: memberInfo.Name,
		ReporterVip:  memberInfo.Vip,
		Category:     msgInfo.Category,
		MatchId:      msgInfo.MatchId,
		Venue:        mdata.GetVenueCnNameByCode(matchData.Venue),
		League:       matchData.League,
		Home:         matchData.Home,
		Away:         matchData.Away,
		ReportReason: req.Reason,
		MsgCreatedAt: msgInfo.Timestamp,
		CreatedAt:    time.Now().Format(utils.TimeBarFormat),
		UpdatedAt:    time.Now().Format(utils.TimeBarFormat),
	}

	err = sqldb.CreateMsgReportData(insertMsgReportData)
	if err != nil {
		c.Errorf("Report CreateMsgReportData error|data=>%#v|err=>%v", insertMsgReportData, err)
		base.WebRsp(c, utils.ErrInternal, nil, "写入举报记录异常")
		return
	}

	//入库成功: 1.更新该会员的当天举报次数  2.将该会员加入该消息的举报会员集合中  3.判断是否需要推送预警消息 (一条消息只推送一次预警)  3.若成功推送预警消息，需将该消息标识为已推送)
	err = core.SetExpireKV(memberDailyReportCountRedisKey, strconv.Itoa(memberReportCountToday+1), utils.EndOfDay(time.Now()).Sub(time.Now()))
	if err != nil {
		c.Errorf("Report redis set memberDailyReportCount error|key=>%v|result=>%v|err=>%v", memberDailyReportCountRedisKey, memberReportCountToday+1, err)
	}

	ttl := core.GetTTL(fmt.Sprintf(mdata.ChatMsgReportedRedisKey, metadata.GetSiteIdString(c), req.Seq))
	sAdd, err := core.SAdd(fmt.Sprintf(mdata.ChatMsgReportedRedisKey, metadata.GetSiteIdString(c), req.Seq), memberInfo.Name)
	if err != nil {
		c.Errorf("Report redis SAdd error|key=>%v|res=>%v|err=>%v", fmt.Sprintf(mdata.ChatMsgReportedRedisKey, metadata.GetSiteIdString(c), req.Seq), sAdd, err)
	}

	//这里考虑把redis的有效期以开赛时间为准，设为5小时，原因是各种比赛通常有没这么长时间，留5小时足够， 而比赛结束后，对应聊天室也就没了， 再保留这个标识数据也就没太大意义，让其自动失效
	matchTime := time.Now()
	if len(matchData.LiveDate) > 0 {
		tmpTime, err := time.Parse(utils.TimeBarFormat, matchData.LiveDate)
		if err == nil {
			matchTime = tmpTime
		}
	}
	//计算redis有效期: 如果算下来的过期时间早于当前时间 ， 则只给2小时
	redisDuration := matchTime.Add(time.Hour * 5).Sub(time.Now())
	if int(redisDuration.Seconds()) < 1 {
		redisDuration = time.Hour * 2
	}
	if ttl.Seconds() <= 0 {
		err = core.SetExpireKey(fmt.Sprintf(mdata.ChatMsgReportedRedisKey, metadata.GetSiteIdString(c), req.Seq), redisDuration)
		if err != nil {
			c.Errorf("Report redis SetExpireKey error|key=>%v|err=>%v", fmt.Sprintf(mdata.ChatMsgReportedRedisKey, metadata.GetSiteIdString(c), req.Seq), err)
		}
	}

	warnConfig := config.GetChatMsgReportWarnConfig()
	//获取该条消息是否推送预警标识
	warnFlagString, err := core.GetKey(false, fmt.Sprintf(mdata.ChatMsgWarnFlagRedisKey, metadata.GetSiteIdString(c), req.Seq))
	warnFlag, _ := strconv.Atoi(warnFlagString)
	if len(warnFlagString) < 1 {
		c.Errorf("Report redis get warn flag error|key=>%s|result=>%v|err=>%v", fmt.Sprintf(mdata.ChatMsgWarnFlagRedisKey, metadata.GetSiteIdString(c), req.Seq), warnFlagString, err)

		//非redis nil的错误， 默认不推送， 不然的话当redis请求不通， 就会疯狂推
		if err != nil && err != core.RedisNil {
			warnFlag = 1
		}
	}

	//获取该条消息被举报的次数
	reportedCount, err := core.SCard(fmt.Sprintf(mdata.ChatMsgReportedRedisKey, metadata.GetSiteIdString(c), req.Seq))
	c.Infof("Report redis get msg reported count result|key=>%v|count=>%v|err=>%v", fmt.Sprintf(mdata.ChatMsgReportedRedisKey, metadata.GetSiteIdString(c), req.Seq), reportedCount, err)

	//获取后台配置的触发预警的举报次数
	triggerCountString, err := core.GetKey(false, fmt.Sprintf(mdata.TriggerCountRedisKey, metadata.GetSiteIdString(c)))
	triggerCount, _ := strconv.Atoi(triggerCountString)
	if len(triggerCountString) < 1 || triggerCount < 1 {
		c.Errorf("Report redis get warn trigger count err|key=>%v|result=>%v|err=>%v", fmt.Sprintf(mdata.TriggerCountRedisKey, metadata.GetSiteIdString(c)), triggerCountString, err)
		triggerCount = mdata.DefaultWarnCount
	}

	c.Infof("Report push warn msg params|seq=>%v|switch=>%v|reportedCount=>%v|triggerCount=>%v|warnFlag=>%v", req.Seq, warnConfig.Switch, int(reportedCount), triggerCount, warnFlag)

	//推送预警的条件： 配置的预警开关开启 & 该消息被举报的次数达到配置的预警举报次数 && 该消息没被举报过
	if warnConfig.Switch && int(reportedCount) >= triggerCount && warnFlag < 1 {
		msgStr := fmt.Sprintf(mdata.ChatMsgReportTemplate,
			metadata.GetSiteIdString(c),
			warnConfig.Platform,
			insertMsgReportData.Venue,
			insertMsgReportData.Home+"VS"+insertMsgReportData.Away,
			memberInfo.Name,
			msgInfo.Name,
			fmt.Sprintf("VIP%v", msgInfo.Vip),
			msgInfo.Msg,
			msgInfo.Timestamp)
		res, err := send2bot(warnConfig.ServiceCode, msgStr, 3)
		c.Infof("Report send2bot response|seq=>%s|msg=>%s|res=>%v|err=>%v", req.Seq, msgInfo.Msg, res, err)
		if err == nil {
			//推送成功后， 将推送标识置为已推送 (这里考虑把redis的有效期以开赛时间为准，设为5小时，原因是各种比赛通常有没这么长时间，留5小时足够， 而比赛结束后，对应聊天室也就没了， 再保留这个标识数据也就没太大意义，让其自动失效)
			ttl = core.GetTTL(fmt.Sprintf(mdata.ChatMsgWarnFlagRedisKey, metadata.GetSiteIdString(c), req.Seq))
			if ttl.Seconds() <= 0 {
				err = core.SetExpireKV(fmt.Sprintf(mdata.ChatMsgWarnFlagRedisKey, metadata.GetSiteIdString(c), req.Seq), "1", redisDuration)
				if err != nil {
					c.Errorf("Report  redis SetExpireKV err|key=>%v|err=>%v", fmt.Sprintf(mdata.ChatMsgWarnFlagRedisKey, metadata.GetSiteIdString(c), req.Seq), err)
				}
			}
		}
	}

	base.WebRsp(c, utils.StatusOK, nil, utils.MsgSuccess)
	return
}

// 推送预警消息
func send2bot(serviceCode, msg string, botType int) (string, error) {
	url := config.GetApplication().VerifyCodeDomain + "/verifycode/bot/v1/send"
	params := mdata.MustMarshal(map[string]interface{}{
		"serviceCode": serviceCode,
		"botType":     botType,
		"msg":         msg,
	})

	data, err := httpclient.POSTJson(
		url,
		params,
		map[string]string{"Content-Type": "application/json", mdata.HeaderSite: "9999"},
		nil,
	)
	return string(data), err
}

// 获取举报原因
func GetReportReasons(c *context.Context) {
	base.WebRsp(c, utils.StatusOK, mdata.ReportReasons, utils.MsgSuccess)
	return
}

func RedPackageRainHandler(c *context.Context) {
	req := new(mdata.ResEnvelopReq)
	period1 := time.Now()
	err := mdata.Cjson.NewDecoder(c.Request.Body).Decode(req)
	if err != nil {
		if err == io.EOF {
			c.Errorf("红包雨处理，耗时：%.2f秒", time.Since(period1).Seconds())
			period2 := time.Now()
			base.WebRsp(c, utils.ErrInternal, nil, "读取红包雨输入失败")
			c.Infof("红包雨处理，输出响应耗时：%.2f秒", time.Since(period2).Seconds())
		} else {
			c.Errorf("RedPackageRainHandler error: %v", err)
			base.WebRsp(c, utils.ErrInternal, nil, "读取红包雨输入失败")
		}
		return
	}
	pkg := ws.NewPacket()
	pkg.Write(mdata.MustMarshal(req))
	ws.SendMsgChan(pkg)
}

// GetOnlineUser 获取在线用户
func GetOnlineUser(c *context.Context) {
	req := new(mdata.OnlineUserReq)
	start := time.Now()

	// 解析请求
	if err := mdata.Cjson.NewDecoder(c.Request.Body).Decode(req); err != nil {
		if err == io.EOF {
			c.Errorf("获取在线用户，耗时：%.2f秒", time.Since(start).Seconds())
			base.WebRsp(c, utils.ErrInternal, nil, "获取在线用户失败")
		} else {
			c.Errorf("获取在线用户 error: %v", err)
			base.WebRsp(c, utils.ErrInternal, nil, "获取在线用户失败")
		}
		return
	}

	var result any

	switch req.Type {
	case 1: // 单个用户
		siteId := strconv.Itoa(req.SiteId)
		client := app.GetApp().GetClient(req.UserName, siteId)
		if client == nil {
			base.WebRsp(c, utils.ErrNotFound, nil, "用户不存在")
			return
		}

		member, err := client.Member()
		if err != nil {
			base.WebRsp(c, utils.ErrInternal, nil, "获取用户失败")
			return
		}

		result = &mdata.OnlineUserRes{
			UserName: member.Name,
			SiteId:   client.GetSiteId(),
			Info:     member,
			Hub:      client.GetHub().GetHubID(),
		}

	case 2: // 全部用户（分页）
		allClients := app.GetApp().GetAllClient()
		clients := app.GetApp().Paginate(allClients, req.PageNum, req.PageSize)

		var out []*mdata.OnlineUserRes
		for _, client := range clients {
			if client == nil {
				continue
			}
			member, err := client.Member()
			if err != nil {
				continue
			}
			out = append(out, &mdata.OnlineUserRes{
				UserName: member.NickName,
				SiteId:   client.GetSiteId(),
				Info:     member,
				Hub:      client.GetHub().GetHubID(),
			})
		}
		result = out

	case 3: // 指定 Hub
		clients := app.GetApp().GetClientsByRoomHub(req.Hub)

		var out []*mdata.OnlineUserRes
		for _, client := range clients {
			if client == nil {
				continue
			}
			member, err := client.Member()
			if err != nil {
				continue
			}
			out = append(out, &mdata.OnlineUserRes{
				UserName: member.NickName,
				SiteId:   client.GetSiteId(),
				Info:     member,
				Hub:      client.GetHub().GetHubID(),
			})
		}
		result = out

	default:
		base.WebRsp(c, utils.ErrAPIFailed, nil, "不支持的查询类型")
		return
	}

	// 成功响应
	base.WebRsp(c, utils.StatusOK, result, utils.MsgSuccess)
}

func ValidToken(c *context.Context) {

	req := new(mdata.ValidTokenReq)
	start := time.Now()

	// 解析请求
	if err := mdata.Cjson.NewDecoder(c.Request.Body).Decode(req); err != nil {
		if err == io.EOF {
			c.Errorf("解析token出错，耗时：%.2f秒", time.Since(start).Seconds())
			base.WebRsp(c, utils.ErrInternal, nil, "解析token出错")
		} else {
			c.Errorf("解析token出错 error: %v", err)
			base.WebRsp(c, utils.ErrInternal, nil, "解析token出错")
		}
		return
	}

	loginPars := mdata.LoginReqSchema{Token: req.Token, Body: req.Body}

	sub, err := mdata.ParserToken(&loginPars)
	if (err != nil || sub == nil) && c.Request.Method != "POST" {
		base.WebRsp(c, utils.ErrTokenInvalid, nil, "解析token出错")
		return
	}

	// 成功响应
	base.WebRsp(c, utils.StatusOK, sub, utils.MsgSuccess)

}

func GetDeviceToken(c *context.Context) {

	var req mdata.GetUserDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		base.WebRsp(c, utils.ErrInternal, nil, "解析参数出错")
		return
	}

	redisKeyString := rediskey.UserDeviceTokenCacheKey + req.AppKey
	filed := fmt.Sprintf("%d", req.SiteId) + "_" + req.UserId + "_" + "ios"
	c.Infof("GetDeviceToken redisKeys: %v  filed: %v", redisKeyString, filed)
	val, err := redis.HGet(true, redisKeyString, filed)
	if err != nil {

		c.Errorf("GetDeviceToken redis hget error: %v", err)
		base.WebRsp(c, utils.ErrInternal, nil, err.Error())
		return
	}

	// 成功响应
	base.WebRsp(c, utils.StatusOK, val, utils.MsgSuccess)

}

func SendMsg(c *context.Context) {

	req := new(mdata.MsgRequest)
	start := time.Now()

	// 解析请求
	if err := mdata.Cjson.NewDecoder(c.Request.Body).Decode(req); err != nil {
		if err == io.EOF {
			c.Errorf("解析token出错，耗时：%.2f秒", time.Since(start).Seconds())
			base.WebRsp(c, utils.ErrInternal, nil, "解析token出错")
		} else {
			c.Errorf("解析token出错 error: %v", err)
			base.WebRsp(c, utils.ErrInternal, nil, "解析token出错")
		}
		return
	}

	count := req.Count
	msg := req.Msg

	// JSON -> []byte
	data, err := json.Marshal(msg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("binary json: %v\n", data) // []byte
	fmt.Printf("as string : %s\n", string(data))

	for i := 0; i < count; i++ {
		app.GetApp().SendAllClientForMsg(data)
		if i+1 < count {
			time.Sleep(time.Duration(req.Delay) * time.Second) // interval: e.g. 50 * time.Millisecond
		}
	}
	base.WebRsp(c, utils.StatusOK, nil, utils.MsgSuccess)
}
