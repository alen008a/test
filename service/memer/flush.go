package memer

import (
	"fmt"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/db/sqldb"
	"msgPushSite/lib/ws"
	"msgPushSite/mdata"
	"msgPushSite/mdata/rediskey"
	"msgPushSite/service"
	"strings"
	"time"
)

// getVIPConfig 获取后台VIP发言等级 特效等级/状态 信息
func getVIPConfig(c *ws.Context) (*mdata.VIPConf, error) {
	var (
		err error
		res = new(mdata.VIPConf)
	)
	vipConfByte, err := core.GetKeyBytes(false, fmt.Sprintf(rediskey.ChatVIPConf, c.SiteId))
	if err != nil && err != core.RedisNil {
		c.Errorf("BroadcastVerifyService GetKeyBytes is error: %s", err.Error())
		return nil, mdata.SerViceStatusErr
	}
	err = mdata.Cjson.Unmarshal(vipConfByte, res)
	if err != nil {

		//TODO 临时注释 需要还原
		//c.Errorf("getVIPConfigFromCache Unmarshal is error: %s", err.Error())
		return nil, mdata.SerViceStatusErr
	}
	//TODO 临时注释 需要还原
	//glog.Infof("getVIPConfig get redis vipConf:%s", string(vipConfByte))
	return res, nil
}

// getRoomInfo 获取当前房间状态
func getRoomInfo(c *ws.Context, siteId, roomId string) (*mdata.LiveMatch, error) {
	var (
		err error
		res = new(mdata.LiveMatch)
	)
	res, err = service.GetRoomInfo(siteId, roomId)
	if err != nil {
		c.Errorf("getRoomInfoFromCache error:%+v | room:[%s] | siteId:[%s] | connection id: %s | ip: %s | username: %s ",
			err,
			roomId,
			siteId,
			c.Id,
			c.Ip(),
			c.GetUsername(),
		)
		return nil, mdata.SerViceStatusErr
	}
	return res, nil
}

func getHistoryRecordByRoomIDFromCache(c *ws.Context, siteId, roomID string) ([]*mdata.BroadcastRoomRspSchema, bool, error) {
	var (
		total int64
		err   error
		res   []*mdata.BroadcastRoomRspSchema
	)
	if strings.Count(roomID, "_") >= 2 {
		roomID = roomID[strings.Index(roomID, "_")+1:]
	}
	key := fmt.Sprintf(rediskey.LiveMatchMessage, siteId, roomID)
	history, err := core.LRange(false, key, 0, 20)
	if err != nil {
		c.Errorf("getHistoryRecordByRoomIDFromCache LRange is error: %s", err.Error())
		return res, false, err
	}
	cnt := len(history)
	for i := 1; i <= cnt; i++ {
		data := []byte(history[cnt-i])
		if len(data) == 0 {
			continue
		}
		ele := new(mdata.BroadcastRoomRspSchema)
		err = mdata.Cjson.Unmarshal(data, ele)
		if err != nil {
			continue
		}

		//查询消息是否被自己举报过
		username := c.GetUsername()
		if len(ele.Seq) > 0 && len(username) > 0 {
			reported, err := sqldb.CheckMsgReportedOrNot(siteId, ele.Seq, username)
			if err != nil {
				c.Errorf("ws getHistoryRecordByRoomIDFromCache CheckMsgReportedOrNot err|seq=>%v|username=>%v|err=>%v", ele.Seq, username, err)
			}

			if err == nil {
				if reported {
					//被自己举报过
					ele.IsReported = 1
				} else {
					//没被自己举报过， 允许举报
					ele.AllowReport = 1
				}
			}
		}
		res = append(res, ele)
	}
	total, err = core.LLen(key, false)
	if err != nil {
		c.Errorf("getHistoryRecordByRoomIDFromCache LLen is error: %s", err.Error())
		return res, false, err
	}
	var next bool
	if total > 20 {
		next = true
	}
	return res, next, nil
}

// getAllRoomStatus 获取全局房间状态 false 正常开启/ true 所有房间关闭
func getAllRoomStatus(siteId string) (bool, error) {
	res, err := core.KeyExist(false, fmt.Sprintf(rediskey.AllMatchBan, siteId))
	if err != nil {
		return false, mdata.SerViceStatusErr
	}
	return res, nil
}

// getRoomStatus 获取房间状态
func getRoomStatus(c *ws.Context, siteId, roomID string) error {
	// 1. 判断全局房间是否在维护
	status, err := getAllRoomStatus(siteId)
	if err != nil {
		return mdata.GlobalRoomStatusErr
	}
	if status {
		return mdata.AllRoomMaintainErr
	}
	// 2. 获取当前房间信息
	room, err := getRoomInfo(c, siteId, roomID)
	if err != nil {
		return err
	}
	// 3. 判断赛事是否开启
	//date, err := utils.ParseTime(room.LiveDate)
	//if err != nil {
	//	return mdata.SerViceStatusErr
	//}
	//if !time.Now().After(date) {
	//	return mdata.RoomNotStartErr
	//}
	// 4. 判断当前房间状态
	if room.Status == 2 {
		return mdata.RoomStatusErr
	}
	return nil
}

func SetLock(key string, expire time.Duration) (bool, error) {
	return core.SetNX(key, "1", expire)
}

func getShareBigBet(siteId string, amount float64) bool {

	bigShare, err := core.GetKeyFloat(false, fmt.Sprintf(rediskey.ShareBetsBigAmountLimit, siteId))
	if err != nil {
		return false
	}

	if amount >= bigShare {
		return true
	}
	return false
}

func getBulletSetting(siteId string) (bool, bool) {

	button, err := core.KeyExist(false, fmt.Sprintf(rediskey.BulletButtonSetting, siteId))
	if err != nil {
		return true, true
	}
	show, err := core.KeyExist(false, fmt.Sprintf(rediskey.BulletShowSetting, siteId))
	if err != nil {
		return true, true
	}
	return !button, !show
}
