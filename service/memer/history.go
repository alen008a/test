package memer

import (
	"fmt"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/internal/context"
	"msgPushSite/internal/glog"
	"msgPushSite/lib/es"
	"msgPushSite/lib/ws"
	"msgPushSite/mdata"
	"msgPushSite/mdata/rediskey"
	"msgPushSite/service"
	"strconv"
	"strings"
	"time"
)

func GetHistoryRecord(c *ws.Context, req *mdata.HistoryRecordReqSchema) ([]*mdata.BroadcastRoomRspSchema, int64, error) {
	info, _ := c.Member()
	if req.PageNum == 0 {
		req.PageNum = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}
	req.SiteId = info.SiteId
	if req.ChatCategory == 2 {
		useCache := len(req.BeginTime) < 1 && len(req.EndTime) < 1 && req.PageNum == 1
		//查询当前时间内一个小时的，先查30秒的缓存，没有再查db
		if useCache {
			data, count, err := GetLiveMatchMsgRoom(c, info.SiteId, req.RID, strconv.Itoa(req.CategoryType))
			if err == nil {
				c.Infof("GetLiveMatchMsgRoom success: %s", req.RID)
				return data, count, nil
			}
		}
		//查db
		var (
			res   []*mdata.BroadcastRoomRspSchema
			err   error
			total int64
		)
		res, total, err = es.GetHistoryByIDFromES(req, "")
		if err != nil {
			c.Error("GetHistoryRecord GetBetHistoryByID is error: %s", err.Error())
			return nil, 0, mdata.SerViceStatusErr
		}
		//查第一页 不查数据库， 数量>=PageSize, 默认total为2页 否则为 1页
		if useCache {
			c.Infof("GetLiveMatchMsgRoom roomed:%s,total: %s", req.RID, len(res))
			if len(res) >= req.PageSize {
				total = int64(req.PageSize + 1)
			} else {
				total = int64(len(res))
			}
			//查询当前时间内一个小时的，设置30秒的缓存
			err = SetLiveMatchMsgRoom(req.SiteId, req.RID, strconv.Itoa(req.CategoryType), res, total)
			if err != nil {
				c.Error("SetLiveMatchMsgRoom is Marshal error: %v", err)
				return nil, 0, err
			}
			return nil, 0, mdata.SerViceStatusErr
		}
		return res, total, nil
	}

	var (
		lastTimeStamp string
		err           error
	)
	lastTimeStamp, err = es.GetHistoryTimestampBySeqFromES(req)

	if err != nil {
		c.Error("GetHistoryRecord GetHistoryIDBySeq is error: %s", err.Error())
		return nil, 0, mdata.SerViceStatusErr
	}

	var (
		res   []*mdata.BroadcastRoomRspSchema
		total int64
	)
	res, total, err = es.GetHistoryByIDFromES(req, lastTimeStamp)
	if err != nil {
		c.Error("GetHistoryRecord GetHistoryByID is error: %s", err.Error())
		return nil, 0, mdata.SerViceStatusErr
	}
	return res, total, nil
}

func SetLiveMatchMsgRoom(siteId, roomId string, categoryType string, data []*mdata.BroadcastRoomRspSchema, count int64) error {
	key := fmt.Sprintf(rediskey.DoubleLiveMatchMsg, siteId, roomId, categoryType)
	obj, err := mdata.Cjson.MarshalToString(data)
	if err != nil {
		return err
	}
	value := obj + "|" + strconv.FormatInt(count, 10)
	err = service.SetDoubleCache(key, value, 30*time.Second)
	if err != nil {
		return err
	}
	return nil
}

func GetLiveMatchMsgRoom(c *ws.Context, siteId, roomId string, categoryType string) (obj []*mdata.BroadcastRoomRspSchema, count int64, error error) {
	defer func() {
		if err := recover(); err != nil {
			glog.Emergency("GetLiveMatchMsgRoom panic recover error|err=>%v", err)
		}
	}()
	error = mdata.SerViceStatusErr
	key := fmt.Sprintf(rediskey.DoubleLiveMatchMsg, siteId, roomId, categoryType)
	data, err := service.GetDoubleCache(key)
	if err != nil {
		c.Error("GetLiveMatchMsgRoom is GetDoubleCache error: %s", err.Error())
		return obj, count, error
	}
	if data == "" {
		c.Error("GetLiveMatchMsgRoom is null")
		return obj, count, error
	}
	dataArray := strings.Split(data, "|")
	if len(dataArray) != 2 {
		c.Error("GetLiveMatchMsgRoom is Split error: %v", dataArray)
		return obj, count, error
	}
	count, err = strconv.ParseInt(dataArray[1], 10, 64)
	if count == 0 {
		error = nil
		return obj, count, error
	}
	err = mdata.Cjson.UnmarshalFromString(dataArray[0], &obj)
	if err != nil {
		c.Error("GetLiveMatchMsgRoom is Unmarshal error: %s", err.Error())
		return obj, count, error
	}
	error = nil
	return obj, count, error
}

// GetHistoryRecords http 协议获取聊天室记录
func GetHistoryRecords(c *context.Context, req *mdata.HistoryRecordReqSchema) ([]*mdata.BroadcastRoomRspSchema, int64, error) {
	var (
		res   []*mdata.BroadcastRoomRspSchema
		err   error
		total int64
	)

	if req.PageNum == 0 {
		req.PageNum = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}
	//晒单逻辑
	if req.ChatCategory == 2 {
		useCache := len(req.BeginTime) < 1 && len(req.EndTime) < 1 && req.PageNum == 1 && req.Seq == ""
		//查询当前时间内一个小时的，先查30秒的缓存，没有再查db
		if useCache {
			data, count, err := GetLiveMatchRoomMessages(c, req.SiteId, req.RID, strconv.Itoa(req.CategoryType))
			if err == nil {
				c.Infof("GetHistoryRecords sucess: %s", req.RID)
				return data, count, nil
			}
		}
		res, total, err = getHistoryData(c, req)
		if err != nil {
			c.Error("GetHistoryRecords getHistoryData error: %s", err.Error())
			return nil, 0, mdata.SerViceStatusErr
		}
		//查第一页 不查数据库， 数量>=PageSize, 默认total为2页 否则为 1页
		if useCache {
			c.Infof("GetHistoryRecords roomed:%s,total: %s", req.RID, len(res))
			if len(res) >= req.PageSize {
				total = int64(req.PageSize + 1)
			} else {
				total = int64(len(res))
			}
			//查询当前时间内一个小时的，设置30秒的缓存
			err = SetLiveMatchMsgRoom(req.SiteId, req.RID, strconv.Itoa(req.CategoryType), res, total)
			if err != nil {
				c.Error("SetLiveMatchMsgRoom is Marshal error: %s", err.Error())
				return nil, 0, err
			}
		}
		return res, total, nil
	} else { //其他聊天类型
		//第一页走缓存
		if req.PageNum == 1 && req.Seq == "" {
			res, total, err = getFirstPageRecordFromCache(c, req.SiteId, req.RID)
			if err == nil && len(res) > 0 && total > 0 {
				return res, total, nil
			} else {
				if err != nil {
					c.Error("GetHistoryRecords getFirstPageRecordFromCache error: %v", err)
				}
			}
		}

		res, total, err = getHistoryData(c, req)
		if err != nil {
			c.Error("GetHistoryRecords getHistoryData error: %s", err.Error())
			return nil, 0, mdata.SerViceStatusErr
		}
		return res, total, nil
	}
}

func getHistoryData(c *context.Context, req *mdata.HistoryRecordReqSchema) (res []*mdata.BroadcastRoomRspSchema, total int64, err error) {

	var (
		lastTimeStamp string
	)
	lastTimeStamp, err = es.GetHistoryTimestampBySeqFromES(req)

	if err != nil {
		c.Error("GetHistoryRecord GetHistoryIDBySeq is error: %s", err.Error())
		return nil, 0, mdata.SerViceStatusErr
	}

	res, total, err = es.GetHistoryByIDFromES(req, lastTimeStamp)
	if err != nil {
		c.Error("GetHistoryRecord GetHistoryByID is error: %s", err.Error())
		return nil, 0, mdata.SerViceStatusErr
	}

	return res, total, nil
}

func GetLiveMatchRoomMessages(c *context.Context, siteId, roomId string, categoryType string) (obj []*mdata.BroadcastRoomRspSchema, count int64, error error) {
	error = mdata.SerViceStatusErr
	key := fmt.Sprintf(rediskey.DoubleLiveMatchMsg, siteId, roomId, categoryType)
	data, err := service.GetDoubleCache(key)
	if err != nil {
		c.Error("GetLiveMatchMsgRoom is GetDoubleCache error: %s", err.Error())
		return obj, count, error
	}
	if data == "" {
		c.Error("GetLiveMatchMsgRoom is null")
		return obj, count, error
	}
	dataArray := strings.Split(data, "|")
	if len(dataArray) != 2 {
		c.Error("GetLiveMatchMsgRoom is Split error: %v", dataArray)
		return obj, count, error
	}
	count, err = strconv.ParseInt(dataArray[1], 10, 64)
	if count == 0 {
		error = nil
		return obj, count, error
	}
	err = mdata.Cjson.UnmarshalFromString(dataArray[0], &obj)
	if err != nil {
		c.Error("GetLiveMatchMsgRoom is Unmarshal error: %s", err.Error())
		return obj, count, error
	}
	error = nil
	return obj, count, error
}

// 获取消息记录第一页缓存数据
func getFirstPageRecordFromCache(c *context.Context, siteId, roomID string) ([]*mdata.BroadcastRoomRspSchema, int64, error) {
	var (
		total int64
		err   error
		res   []*mdata.BroadcastRoomRspSchema
	)
	key := fmt.Sprintf(rediskey.LiveMatchMessage, siteId, roomID)
	history, err := core.LRange(false, key, 0, 20)
	if err != nil {
		c.Errorf("getFirstPageRecordFromCache LRange is error: %v", err)
		return res, 0, err
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
		res = append(res, ele)
	}
	//从redis中获取消息总数 (消息入库脚本服务中写入的， 这里只是取)
	totalMsgCountKey := fmt.Sprintf(rediskey.LiveTotal, siteId, roomID)
	totalString, err := core.GetKey(false, totalMsgCountKey)
	if err != nil {
		c.Errorf("getFirstPageRecordFromCache get total msg count error: %v", err)
		return res, 0, err
	}
	totalString2Int, _ := strconv.Atoi(totalString)
	total = int64(totalString2Int)
	return res, total, nil
}
