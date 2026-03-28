package service

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/db/sqldb"
	"msgPushSite/internal/glog"
	"msgPushSite/lib/cache"
	"strings"

	"msgPushSite/mdata"
	"msgPushSite/mdata/rediskey"

	"msgPushSite/utils"
	"time"
)

func SetDoubleCache(key string, value string, expire time.Duration) error {
	cache.GSet(key, value, expire)
	err := core.SetExpireKV(key, value, expire)
	if err != nil {
		return err
	}
	return nil
}

func GetDoubleCache(key string) (string, error) {
	d, ok := cache.GGet(key)
	if !ok {
		resp, err := core.GetKey(false, key)
		if err != nil {
			return "", err
		}
		return resp, nil
	}
	result, _ := d.(string)
	return result, nil
}

// GetRoomInfo 获取当前房间状态
func GetRoomInfo(siteId, roomId string) (*mdata.LiveMatch, error) {
	var (
		err error
		res = new(mdata.LiveMatch)
	)
	if strings.Count(roomId, "_") >= 2 {
		roomId = roomId[strings.Index(roomId, "_")+1:]
	}
	rInfoByte, err := core.GetKeyBytes(false, fmt.Sprintf(rediskey.LiveMatch, siteId, roomId))

	if err != nil && err != core.RedisNil {
		glog.Errorf("getRoomInfo GetKeyBytes is error: %s", err.Error())
		return nil, err
	}
	if err == core.RedisNil || len(rInfoByte) == 0 {
		res, err = sqldb.GetMatchDataByMatchID(siteId, roomId)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			glog.Errorf("getRoomInfo GetLiveMatchByMatchID is error: %s", err.Error())
			return nil, err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) || res == nil {
			res = &mdata.LiveMatch{
				MatchId:  roomId,
				Status:   1,
				LiveDate: time.Now().Format(utils.TimeBarFormat),
			}
		} else {
			err = flush2StringCache(res, fmt.Sprintf(rediskey.LiveMatch, siteId, roomId))
			if err != nil {
				return nil, err
			}
		}
	} else {
		err = mdata.Cjson.Unmarshal(rInfoByte, res)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func flush2StringCache(data interface{}, key string) error {
	dataByte, err := mdata.Cjson.Marshal(data)
	if err != nil {
		return mdata.SerViceStatusErr
	}
	err = core.SetExpireKV(key, string(dataByte), 1*time.Hour)
	if err != nil {
		return mdata.SerViceStatusErr
	}
	return nil
}
