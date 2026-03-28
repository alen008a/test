package memer

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"msgPushSite/config"
	"msgPushSite/db/sqldb"
	"msgPushSite/lib/cache"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"

	"msgPushSite/db/redisdb/core"
	"msgPushSite/internal/glog"
	"msgPushSite/lib/ws"
	"msgPushSite/mdata"
	"msgPushSite/mdata/rediskey"
	"msgPushSite/utils"
)

var (
	DeadLineTime, _ = time.Parse(utils.TimeBarFormat, "2024-05-01 00:00:00")
)

// AuthJoinRoomService 1.登陆连接获取用户状态 -》2. 获取房间状态以及是否开启特效 -》3. a.单播房间状态以及聊天历史记录 b.房间广播进入房间
func AuthJoinRoomService(c *ws.Context, roomID string, info *mdata.MemberInfo) (*mdata.AuthJoinRoomSelfRspSchema, error) {
	var (
		err  error
		self = new(mdata.AuthJoinRoomSelfRspSchema)
	)

	// 1. 检查房间状态
	err = getRoomStatus(c, info.SiteId, roomID)
	if err != nil {
		return nil, err
	}
	// 2. 获取单播数据
	history, next, err := getHistoryRecordByRoomIDFromCache(c, info.SiteId, info.SiteId+"_"+roomID)
	if err != nil {
		return nil, err
	}
	// 3. 加入房间基本信息推送
	room, err := getRoomInfo(c, info.SiteId, roomID)
	if err != nil {
		return nil, err
	}

	vip, err := getVIPConfig(c)
	if err != nil {
		return nil, err
	}

	status := GetIsBan(info.SiteId, info.Name)
	self.SpeechStatus = 0
	if status {
		self.SpeechStatus = 1
	}
	notify, speechStatus := MemberBannedStatus(info.SiteId, info.Name, self.SpeechStatus)
	self.SpeechStatus = speechStatus
	self.Notify = notify
	self.VipMin = vip.VipMin
	self.VipMax = vip.VipMax
	self.EffectMin = vip.EffectMin
	self.EffectMax = vip.EffectMax
	self.BulletButton, self.BulletOpen = getBulletSetting(info.SiteId)
	self.History = history
	self.LiveDate = room.LiveDate
	self.LiveStatus = room.Status
	self.Next = next
	self.EffectOpen = vip.EffectOpen
	self.BsOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.BetsStrategiesOpen, info.SiteId))
	self.AdvOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.AdvStrategiesOpen, info.SiteId))
	self.GiftOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.LiveGiftStatusOpen, info.SiteId))
	self.CopyOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.ChatIsCanCopyContentOpen, info.SiteId))
	self.ScoreOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.ChatMatchScoreStatusOpen, info.SiteId))
	// 4. 更新缓存
	var oldRID string
	if oldRoom := c.GetRoom(); oldRoom != nil {
		oldRID = oldRoom.GetID()
		// 切换到不同的房间
		if !strings.Contains(oldRID, roomID) {
			ChangeRoom(c, info.Name, oldRID)
		}
	}
	// 5. 必须要在redis切换之后
	ws.JoinRoom(c.Client, roomID, room.LiveDate)
	// 6. 检测当前会员加入房间是否要提示语广播
	if err = PromptJoinRoom(c, nil); err != nil {
		return nil, err
	}
	return self, nil
}

// 清空本地禁言缓存
func ClearBannedLocalCache(siteId string) {
	cache.DeleteCache(fmt.Sprintf(rediskey.MemberSpeechStatus, siteId))
	cache.DeleteCache(fmt.Sprintf(rediskey.MemberSpeechBannedDuration, siteId))
}

// 禁言名单 本地缓存
func GetIsBan(siteId, name string) bool {
	nameList, err := cache.GetOrSet(fmt.Sprintf(rediskey.MemberSpeechStatus, siteId), 5*time.Minute, func() (i interface{}, e error) {
		return core.SMembers(false, fmt.Sprintf(rediskey.MemberSpeechStatus, siteId))
	})
	if err != nil {
		glog.Errorf("查询MemberSpeechStatus失败:%+v", err)
		return false
	}
	nList, ok := nameList.([]string)
	if ok {
		for _, value := range nList {
			if name == value {
				return true
			}
		}
		return false
	}
	glog.Errorf("查询MemberSpeechStatus断言失败:%s", nameList)
	return false
}

// GetBanInfo 禁言信息 本地缓存
func GetBanInfo(siteId, name string) (string, error) {
	banInfoHash, err := cache.GetOrSet(fmt.Sprintf(rediskey.MemberSpeechBannedDuration, siteId), 5*time.Minute, func() (i interface{}, e error) {
		return core.HGetAll(false, fmt.Sprintf(rediskey.MemberSpeechBannedDuration, siteId))
	})
	if err != nil {
		glog.Errorf("查询MemberSpeechBannedDuration失败:%+v", err)
		return "", err
	}
	bInfoHash, ok := banInfoHash.(map[string]string)
	if ok {
		return bInfoHash[name], nil
	}
	glog.Errorf("查询MemberSpeechBannedDuration失败断言失败:%s", bInfoHash)
	return "", nil
}

// 检查用户禁言状态 到期自动解禁
func MemberBannedStatus(siteId, name string, speechStatus int) (string, int) {
	if speechStatus == 0 {
		return "", speechStatus
	}
	bannedInfoStr, err := GetBanInfo(siteId, name)
	if bannedInfoStr != "" {
		var bannedInfo mdata.BannedInfo
		_ = mdata.Cjson.UnmarshalFromString(bannedInfoStr, &bannedInfo)
		if bannedInfo.Duration == -1 {
			return "永久禁言中", speechStatus
		} else {
			// 判断是否到解禁时间
			bannedStart, _ := utils.BjTBarFmtTime(bannedInfo.BannedStart)
			if utils.BJNowTime().After(bannedStart.Add(time.Duration(bannedInfo.Duration) * time.Hour)) {
				//到过期时间
				ctx := context.Background()
				err = core.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
					// 解禁，将会员id从redis的set删除就行了
					err = pipeliner.SRem(ctx, fmt.Sprintf(rediskey.MemberSpeechStatus, siteId), name).Err()
					if err != nil {
						return err
					}
					//删除禁言时长
					err = pipeliner.HDel(ctx, fmt.Sprintf(rediskey.MemberSpeechBannedDuration, siteId), name).Err()
					if err != nil {
						return err
					}
					return nil
				})
				ClearBannedLocalCache(siteId)
				if err != nil {
					glog.Errorf("MemberBannedStatus core.TxPipelined err:%+v param:%+v", err, name)
					return "", speechStatus
				}
				speechStatus = 0
				err = sqldb.Live().Table("members_banned_record").
					Where("site_id = ? and name = ? and deleted = 0", siteId, name).
					Updates(map[string]interface{}{
						"deleted": 1, "admin_name": "SYSTEM",
						"end_time": time.Now().Format(utils.TimeBarFormat),
					}).Error

				if err != nil {
					glog.Errorf("MemberBannedStatus update err:%+v param:%+v", err, name)
				}
				return "", speechStatus
			} else {
				countDown := utils.GetCutDownInterval(bannedStart.Add(time.Duration(bannedInfo.Duration) * time.Hour))
				if countDown == "EOF" {
					return "解禁中,请稍等片刻", speechStatus
				}
				return fmt.Sprintf("禁言中,%s后解除", countDown), speechStatus
			}
		}
	}
	return "", speechStatus
}

// 返回VIP等级对应的Ip服务地址
func IpServerAddress(siteId, clientType string) []*mdata.IpServer {
	var ipServerConfig = make([]*mdata.IpServer, 0)
	if _, ok := utils.IpServerClientTypeMap[clientType]; !ok {
		return ipServerConfig
	}

	ipServerConfigKey := fmt.Sprintf("IpServerConfig_%s_%s_domain", siteId, clientType)
	// HGetAll key 是依照现有的 VIP 等级是固定数量
	ipConfig, err := core.GetKey(true, ipServerConfigKey)
	if err != nil {
		if err == core.RedisNil {
			glog.Errorf("redisdb.GetKey err=%s| key=%s", err.Error(), ipServerConfigKey)
			return ipServerConfig
		}
	}

	if ipConfig != "" {
		var ipConfigList map[int][]*mdata.IpServer
		if err = mdata.Cjson.UnmarshalFromString(ipConfig, &ipConfigList); err != nil {
			glog.Errorf("get ipConfig unmarshal data | err=%+v | ipconfig=%v", err, ipConfig)
		}

		for _, v := range ipConfigList {
			ipServerConfig = append(ipServerConfig, v...)
		}
	}
	return ipServerConfig
}

// WS 推送加密后的 API 下发 APP 的文件内容
func WSApiApp(siteId, clientType string) (ipServerApps []string) {
	if _, ok := utils.IpServerClientTypeMap[clientType]; !ok {
		return ipServerApps
	}
	// 端口类型 : 1=ios、2=ios_sport、3=android、4=android_sport
	switch clientType {
	case utils.AndroidClientType:
		// AndroidClientType
		clientType = "3"
	case utils.AndroidSportClientType:
		// AndroidSportClientType
		clientType = "4"
	case utils.IOSClientType:
		// IOSClientType
		clientType = "1"
	case utils.IOSSportClientType:
		// IOSSportClientType
		clientType = "2"
	default:
		glog.Errorf("WSApiApp | clientType=%s", clientType)
		return ipServerApps
	}

	cacheKey := fmt.Sprintf("IpServerAppConfig:%s:%s", siteId, clientType)
	data, err := core.GetKeyBytes(true, cacheKey)
	if err != nil && err != core.RedisNil {
		glog.Errorf("WSApiApp | err=%s | key=%s", err.Error(), cacheKey)
		return ipServerApps
	}
	// 防止写入多余日志
	if len(data) > 0 {
		if err = mdata.Cjson.Unmarshal([]byte(data), &ipServerApps); err != nil {
			glog.Errorf("WSApiApp unmarshal | err=%v | data=%v | cacheKey=%s", err, data, cacheKey)
		}
	}
	return ipServerApps
}

// UnAuthJoinRoomService 1.登陆连接获取用户状态 -》2. 获取房间状态 -》3. 单播房间状态以及聊天历史记录
func UnAuthJoinRoomService(c *ws.Context, roomID string) (*mdata.UnAuthJoinRoomRspSchema, error) {
	var (
		err error
		res = new(mdata.UnAuthJoinRoomRspSchema)
	)
	err = getRoomStatus(c, c.GetSiteId(), roomID)
	if err != nil {
		c.Info("UnAuthJoinRoomService getRoomStatus is error: ", err.Error())
		return nil, err
	}

	history, next, err := getHistoryRecordByRoomIDFromCache(c, c.GetSiteId(), roomID)
	if err != nil {
		return nil, err
	}
	room, err := getRoomInfo(c, c.GetSiteId(), roomID)
	if err != nil {
		return nil, err
	}
	vip, err := getVIPConfig(c)
	if err != nil {
		return nil, err
	}
	res.BulletButton, res.BulletOpen = getBulletSetting(c.GetSiteId())
	res.History = history
	res.LiveDate = room.LiveDate
	res.LiveStatus = room.Status
	res.Next = next
	res.EffectOpen = vip.EffectOpen
	res.BsOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.BetsStrategiesOpen, c.GetSiteId()))
	res.AdvOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.AdvStrategiesOpen, c.GetSiteId()))
	res.GiftOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.LiveGiftStatusOpen, c.GetSiteId()))
	res.CopyOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.ChatIsCanCopyContentOpen, c.GetSiteId()))
	res.ScoreOpen, _ = core.KeyExist(true, fmt.Sprintf(rediskey.ChatMatchScoreStatusOpen, c.GetSiteId()))
	ws.JoinRoom(c.Client, roomID, room.LiveDate)
	return res, err
}

// BroadcastVerifyService 分享注单 不判断 禁言状态 & 发言等级
func BroadcastVerifyService(c *ws.Context, info *mdata.MemberInfo, req *mdata.BroadcastRoomReqSchema, msg *ws.Msg) (*mdata.BroadcastRoomRspSchema, error) {
	var (
		err     error
		vipConf = new(mdata.VIPConf)
		res     = new(mdata.BroadcastRoomRspSchema)
		amount  float64
	)
	// 1. 判断当前会员是否加入房间
	room := c.GetRoom()
	if room == nil {
		return nil, mdata.NotJoinRoomErr
	}
	// 2. 先校验当前房间状态
	err = getRoomStatus(c, info.SiteId, room.GetID())
	if err != nil {
		return nil, err
	}

	// 临时修改：禁言后不可晒单
	// 3. 校验当前会员等级是否可以发言
	vipConf, err = getVIPConfig(c)
	if err != nil {
		return nil, err
	}
	// 4. 检测当前会员是否被禁言
	speechStatus := GetIsBan(info.SiteId, info.Name)
	if speechStatus {
		notify, speechStatus := MemberBannedStatus(info.SiteId, info.Name, 1)
		//禁言状态
		if speechStatus == 1 {
			mdata.UserSpeechErr = errors.New(notify)
			return nil, mdata.UserSpeechErr
		}
	}

	//添加注单分享校验
	if msg.MsgId == mdata.MsgIDTypeShareBetRecord {
		//只允许app端晒单,H5也可以晒单
		clientType := strings.ToLower(c.ClientType())
		if clientType == utils.WebClientType || utils.ClientTypeMap[clientType] == "" {
			c.Errorf("客户端不允许晒单|client_type=>%v", clientType)
			return nil, fmt.Errorf("%d|当前客户端不允许晒单！", utils.ErrInvalidParams)
		}
		msgByte := []byte(req.Msg)
		if !json.Valid(msgByte) {
			return nil, fmt.Errorf("%d|注单格式错误！", utils.ErrInvalidParams)
		}
		//由于大于1000，有千分符号等货币分隔符，所以这里先转字符串，过了货币分隔符，然后转数字
		shareAmountStr := mdata.Cjson.Get(msgByte, "strBetAmount").ToString()
		shareAmountStr = strings.Replace(shareAmountStr, ",", "", -1)
		shareAmount, err := strconv.ParseFloat(shareAmountStr, 64)
		if err != nil {
			c.Errorf("注单分享请求失败！name:%+v shareAmountStr:%+v err:%s", info.Name, shareAmountStr, err)
			return nil, fmt.Errorf("%d|注单分享请求失败！", utils.ErrInvalidParams)
		}

		// a. 查看当前分享注单状态，未结算注单不能小于100元
		shareStatus := mdata.Cjson.Get(msgByte, "betResult").ToFloat64()
		amount = shareAmount
		// b. 如果分享金额小于后台配置金额，则提示会员
		if shareStatus == 0 {
			cacheAmount, err := core.GetKeyFloat(false, fmt.Sprintf(rediskey.ShareBetsAmountLimit, info.SiteId))
			if err != nil && err != core.RedisNil {
				c.Errorf("BroadcastVerifyService GetKeyFloat is error: %s", err.Error())
				return nil, mdata.SerViceStatusErr
			}
			if cacheAmount == 0 {
				cacheAmount = 100
			}
			if cacheAmount > shareAmount {
				return nil, fmt.Errorf("%d|请分享大于等于%.2f元的注单", utils.ErrShareBetRecordNotEnough, cacheAmount)
			}
			//已结算注单
		} else if utils.IsIntInArray(int(shareStatus), []int{1, 2, 3}) {
			//按照输赢金额判断
			winAmountStr := mdata.Cjson.Get(msgByte, "strBetWin").ToString()
			winAmountStr = strings.Replace(winAmountStr, ",", "", -1)
			winAmount, err := strconv.ParseFloat(winAmountStr, 64)
			if err != nil {
				c.Errorf("注单分享请求失败！name:%+v shareAmountStr:%+v err:%s", info.Name, shareAmountStr, err)
				return nil, fmt.Errorf("%d|注单分享请求失败！", utils.ErrInvalidParams)
			}
			winAmount = math.Abs(winAmount)
			cacheAmount, err := core.GetKeyFloat(false, fmt.Sprintf(rediskey.SettleShareBetsAmountLimit, info.SiteId))
			if err != nil && err != core.RedisNil {
				c.Errorf("BroadcastVerifyService GetKeyFloat is error: %s", err.Error())
				return nil, mdata.SerViceStatusErr
			}
			amount = winAmount
			if cacheAmount == 0 {
				cacheAmount = 100
			}
			if cacheAmount > winAmount && winAmount != 0 {
				return nil, fmt.Errorf("%d|请分享输/赢金额大于等于%.2f元的注单", utils.ErrShareBetRecordNotEnough, cacheAmount)
			}
		}
		// c. 对当前订单做分享次数限制
		key := fmt.Sprintf(rediskey.ShareBetRecordRepeatLimit, info.SiteId, utils.MD5EncryByByte(msgByte))
		number, err := core.Incr(key)
		if err != nil {
			return nil, mdata.SerViceStatusErr
		}
		if number >= 6 {
			return nil, mdata.ShareRecordRepeatErr
		}
		if number == 1 {
			core.SetExpireKey(key, time.Second*2*60)
		}

	} else {
		if !(vipConf.VipMin <= info.Vip && info.Vip <= vipConf.VipMax) {
			return nil, mdata.UserVIPLevelErr
		}
		// 5. 如果当前VIP等级为0，添加发言频率校验
		if v, ok := c.GetProperty("lastSpeech"); ok {
			last := v.(int64)
			now := time.Now()
			//优化
			timeLimit := config.GetChatMessageLimitConfig().SendTimeLimit
			openFlag := config.GetChatMessageLimitConfig().Switch
			if openFlag && last != 0 && now.Unix()-last < timeLimit {
				return nil, mdata.SpeechFrequencyNormalErr
			}
		}

		msgRune := []rune(req.Msg)
		if len(msgRune) > 150 {
			return nil, mdata.MsgLengthLimitExceededErr
		}
	}

	// 7. 生成result
	siteId, _ := strconv.Atoi(info.SiteId)
	res.SiteId = siteId
	res.Msg = req.Msg
	res.VIP = info.Vip
	res.Nickname = info.NickName
	res.MemberId = info.Id
	res.Timestamp = utils.GetBjNowTime().Format(utils.TimeBarFormat)
	res.Category = 1
	res.Seq = msg.Seq
	if msg.MsgId == mdata.MsgIDTypeShareBetRecord {
		res.Category = 2
		res.CategoryType = 1
		if amount >= 10000 || getShareBigBet(info.SiteId, amount) {
			res.CategoryType = 2
		}
	}
	// 8. 记录会员发言时间，预留
	c.SetProperty("lastSpeech", utils.GetBjNowTime().Unix())

	return res, nil
}

// 晒单参数校验
func verifyShareBetRecordParam(msg, salt, clientType string) (string, error) {
	glog.Errorf("晒单原始信息数据：%v", msg)
	if len(msg) < 32+2 {
		return "", errors.New("晒单参数长度错误")
	}
	paramMd5, realMsg := msg[0:32], msg[32:]
	verifyMd5 := Md5Summary(realMsg, salt)
	if paramMd5 != verifyMd5 {
		glog.Errorf("晒单原始信息数据MD5加密不一致，md5盐：%v,原始md5串：%v,后台MD5加密的串：%v", salt, paramMd5, verifyMd5)
		return "", errors.New("晒单加密参数错误")
	}
	if strings.HasPrefix(realMsg, "!") {
		realMsg = realMsg[1:] + "=="
	} else {
		realMsg = realMsg[2:] + realMsg[0:2]
	}
	byteStr, err := base64.StdEncoding.DecodeString(realMsg)
	if err != nil {
		glog.Errorf("晒单格式错误:%+v", err)
		return "", errors.New("晒单格式错误")
	}
	realMsg = string(byteStr)
	//H5端做了特殊处理,在base64之前做了一次urlEncode编码
	if clientType == utils.H5ClientType {
		realMsg, err = url.QueryUnescape(string(byteStr))
		if err != nil {
			glog.Errorf("h5晒单格式错误:%+v", err)
			return "", errors.New("h5端晒单格式错误")
		}
	}
	r := strings.NewReplacer("|", "\"", "%", "{", "!", "}", "$", "[", "^", "]")
	realMsg = r.Replace(realMsg)
	realMsg, err = convertMessageField(realMsg)
	if err != nil {
		glog.Errorf("晒单数据格式错误:%+v", err)
		return "", errors.New("晒单数据格式错误")
	}
	return realMsg, nil
}

// 消息体转换
func convertMessageField(msg string) (string, error) {
	var betRecordDo mdata.ShareBetRecordDO
	var betRecordParam mdata.ShareBetRecordParam
	err := json.Unmarshal([]byte(msg), &betRecordParam)
	if err != nil {
		return "", err
	}
	obtainStructProperties(&betRecordDo, &betRecordParam)
	marshal, err := json.Marshal(betRecordDo)
	if err != nil {
		return "", err
	}
	return string(marshal), nil
}

// struct赋值
func obtainStructProperties(sr *mdata.ShareBetRecordDO, srp *mdata.ShareBetRecordParam) {
	sr.BetResult = srp.BetResult
	sr.ComboName = srp.ComboName
	sr.CurrencyID = srp.CurrencyID
	sr.StrBetAmount = srp.StrBetAmount
	sr.StrBetWin = srp.StrBetWin
	betList := make([]mdata.BetRecord, 0)
	betRecords := srp.BetList
	for i := 0; i < len(betRecords); i++ {
		var bet mdata.BetRecord
		bet.StrStartTime = betRecords[i].StrStartTime
		bet.PlayID = betRecords[i].PlayID
		bet.SelectionType = betRecords[i].SelectionType
		bet.StrBetResult = betRecords[i].StrBetResult
		bet.CanBet = betRecords[i].CanBet
		bet.StrBetHandcap = betRecords[i].StrBetHandcap
		bet.SportType = betRecords[i].SportType
		bet.StrAwayScore = betRecords[i].StrAwayScore
		bet.MarketID = betRecords[i].MarketID
		bet.IsEu = betRecords[i].IsEu
		bet.StrHomeScore = betRecords[i].StrHomeScore
		bet.StrLGName = betRecords[i].StrLGName
		bet.StrBetOdds = betRecords[i].StrBetOdds
		bet.StrAwayTeam = betRecords[i].StrAwayTeam
		bet.StrBetTypeName = betRecords[i].StrBetTypeName
		bet.StrBetMatchResult = betRecords[i].StrBetMatchResult
		bet.StrBetInfo = betRecords[i].StrBetInfo
		bet.StrHomeTeam = betRecords[i].StrHomeTeam
		bet.SportName = betRecords[i].SportName
		bet.SectionID = betRecords[i].SectionID
		bet.EventID = betRecords[i].EventID
		bet.PeriodID = betRecords[i].PeriodID
		bet.MatchType = betRecords[i].MatchType
		bet.RealHandcap = betRecords[i].RealHandcap
		betList = append(betList, bet)
	}
	sr.BetList = betList
}

func Md5Summary(source, salt string) string {
	m5 := md5.New()
	m5.Write([]byte(source))
	m5.Write([]byte(salt))
	resByte := m5.Sum(nil)
	return hex.EncodeToString(resByte)
}

// ChangeRoom 切换房间,删除旧房间活跃人数
func ChangeRoom(c *ws.Context, name, oldRoomID string) {
	// 1. 删除旧房间在线用户
	if oldRoomID != "" {
		activeTotal := fmt.Sprintf(rediskey.LiveRoomActiveTotal, oldRoomID)
		_, err := core.SRem(activeTotal, name)
		if err != nil && err != core.RedisNil {
			c.Errorf("ChangeRoom Room SRem active user %s is error: %s", activeTotal, err.Error())
		}
	}
}

// CacheAndStat 缓存最近消息,统计在线人数
func CacheAndStat(msg *ws.Msg, res *mdata.BroadcastRoomRspSchema) {
	var flag = 0 //是否命中敏感词
	roomId := msg.RoomId
	if strings.Count(roomId, "_") >= 2 {
		roomId = roomId[strings.Index(roomId, "_")+1:]
	}
	redisKey := fmt.Sprintf(rediskey.LiveMatchMessage, msg.SiteId, roomId)
	if res.Category == 1 { //普通聊天才做敏感词校验
		flag, _ = ws.FilterContent(res.Msg)
	}
	if flag == 0 {
		dataMsg, _ := mdata.Cjson.MarshalToString(res)
		core.LPush(redisKey, dataMsg)
		core.LTrim(redisKey, 0, 100)
		addRoomTTl(redisKey, 5)
	}
	totalMsgCountKey := fmt.Sprintf(rediskey.LiveTotal, msg.SiteId, roomId)
	core.IncrBy(totalMsgCountKey, 1)
	addRoomTTl(totalMsgCountKey, 5)
}

func loginUpdateRoom(c *ws.Context, info *mdata.MemberInfo) error {
	memKey := fmt.Sprintf(rediskey.MemberGlobalCID, info.SiteId, info.Name)
	addRoomTTl(memKey, 12) //延长用户房间过期时间

	_, err := core.SAdd(memKey, c.Id)
	if err != nil && err != core.RedisNil {
		c.Errorf("UserBindClients Room SAdd %s is error: %s", fmt.Sprintf("MEMBER_ALL_CID_%s", info.Name), err.Error())
		return mdata.SerViceStatusErr
	}
	addRoomTTl(memKey, 12) //延长用户房间过期时间
	room := c.GetRoom()
	if room == nil {
		return nil
	}

	memKeyV2 := fmt.Sprintf(rediskey.MemberNameRoomBind, info.SiteId, room.GetID(), info.Name)
	addRoomTTl(memKeyV2, 5) //延长用户房间过期时间

	_, err = core.SAdd(memKeyV2, c.Id)
	if err != nil && err != core.RedisNil {
		c.Errorf("UserBindClients Room SAdd %s is error: %s", fmt.Sprintf("MEMBER_NAME_BIND_%s_%s", room.GetID(), info.Name), err.Error())
		return mdata.SerViceStatusErr
	}
	addRoomTTl(memKeyV2, 5) //延长用户房间过期时间
	return nil
}

func DelLevelClients(siteId, roomId, name, cid string) error {
	//延长账号在房间的缓存
	memKey := fmt.Sprintf(rediskey.MemberNameRoomBind, siteId, roomId, name)
	addRoomTTl(memKey, 5)

	_, err := core.SRem(memKey, cid)
	if err != nil && err != core.RedisNil {
		glog.Errorf("UserBindClients Redis SRem %s is error: %s", memKey, err.Error())
		return mdata.SerViceStatusErr
	}
	addRoomTTl(memKey, 5)
	return nil
}

// DelCloseClient 连接关闭时调用
func DelCloseClient(cli *ws.Client, info *mdata.MemberInfo) error {
	if info == nil || info.Name == "" {
		return errors.New("用户信息不存在")
	}
	//延长账户加入所有房间的缓存
	memKey := fmt.Sprintf(rediskey.MemberGlobalCID, info.SiteId, info.Name)
	addRoomTTl(memKey, 5)
	//移除账户加入所有房间的缓存的当前房间
	_, err := core.SRem(memKey, cli.Id)
	if err != nil && err != core.RedisNil {
		glog.Errorf("UserBindClients Global SRem SAdd %s is error: %s", memKey, err.Error())
		return mdata.SerViceStatusErr
	}
	addRoomTTl(memKey, 5)
	room := cli.GetRoom()
	if !(room != nil && room.GetID() != "") {
		return nil
	}
	//延长账号在当前房间的缓存
	memBindKey := fmt.Sprintf(rediskey.MemberNameRoomBind, info.SiteId, room.GetID(), info.Name)
	addRoomTTl(memBindKey, 5)
	//移除账号在当前房间的缓存的当前房间
	_, err = core.SRem(memBindKey, cli.Id)
	if err != nil && err != core.RedisNil {
		glog.Errorf("UserBindClients Redis SRem %s is error: %s", memBindKey, err.Error())
		return mdata.SerViceStatusErr
	}
	addRoomTTl(memBindKey, 5)
	return nil
}

// 延长用户房间过期时间
func addRoomTTl(redisKey string, timeHour int) {
	limitTime := core.GetTTL(redisKey)
	if limitTime.Seconds() <= 60*5 {
		var err = core.SetExpireKey(redisKey, time.Duration(timeHour)*time.Hour)
		if err != nil {
			glog.Error(err)
		}
	}
}
