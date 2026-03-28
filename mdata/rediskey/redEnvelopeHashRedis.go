package rediskey

import (
	"context"
	"fmt"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/mdata"
	"time"
)

type RedEnvelopeHashRedis struct {
	key string
}

var redEnvelopeRedisKey = func(activityId int64, siteId int) string {
	return fmt.Sprintf(RedEnvelopeKey, activityId, siteId)
}

func NewRedEnvelopeHashRedis(activityId int64, siteId int) *RedEnvelopeHashRedis {
	return &RedEnvelopeHashRedis{
		key: redEnvelopeRedisKey(activityId, siteId),
	}
}

func (r *RedEnvelopeHashRedis) GetActivityEnvelopeValueList(c context.Context) ([]*ActivityEnvelopeValue, error) {

	// 获取 TTL
	ttl, err := core.TTL(c, false, r.key)
	if err != nil {
		//c.Errorf("GetActivityEnvelopeValueList get TTL err: %v", err)
	} else {
		if ttl < 0 {
			switch ttl {
			//case -1:
			//	c.Infof("GetActivityEnvelopeValueList Redis key:%s TTL: 永不过期", r.key)
			//case -2:
			//	c.Infof("GetActivityEnvelopeValueList Redis key:%s TTL: key不存在", r.key)
			//default:
			//	c.Infof("GetActivityEnvelopeValueList Redis key:%s TTL: %v", r.key, ttl)
			}
		} else {
			//c.Infof("GetActivityEnvelopeValueList Redis key:%s TTL: %v", r.key, ttl)
		}
	}

	jsonStr, err := core.HGet(false, r.key, "valueLogs")
	if err != nil {
		return nil, err
	}

	var a []*ActivityEnvelopeValue
	err = mdata.Cjson.Unmarshal([]byte(jsonStr), &a)
	if err != nil {
		return nil, fmt.Errorf("key:%s, filed:%s, err:%+v", r.key, "valueLogs", err)
	}

	return a, nil
}

// 获取红包雨信息
func (r *RedEnvelopeHashRedis) GetActiveEnvelope(c *context.Context, siteId int) *ActivityEnvelopeVo {

	key := fmt.Sprintf(CurrentRedEnvelopeActivityKey, siteId)

	existed, err := core.KeyExist(true, key)
	if err != nil {
		return nil
	}

	if !existed {
		return nil
	}

	jsonStr, err := core.GetKey(true, key)
	if err != nil {
		return nil
	}

	result := &ActivityEnvelopeVo{}
	if err := mdata.Cjson.Unmarshal([]byte(jsonStr), result); err != nil {
		return nil
	}

	now := BJNowTime()
	currTime := TimeToMill(now)

	// 默认设为当前活动类型
	result.ActivityType = 1
	result.CurrentTime = currTime

	// 判断是否为全站红包
	sessionKey := fmt.Sprintf(HasRedEnvelopeSessionKey, siteId)
	if sessionExisted, err := core.KeyExist(true, sessionKey); err != nil {
		return nil
	} else if sessionExisted {
		if sessionVal, err := core.GetKey(true, sessionKey); err == nil && sessionVal == "1" {
			result.ActivityType = 0
		}
	}

	return result
}

func (r *RedEnvelopeHashRedis) GetActivityEnvelope(c context.Context) *ActivityEnvelope {
	// 获取 TTL
	_, err := core.TTL(c, false, r.key)
	if err != nil {
		return nil
	}

	jsonStr, err := core.HGet(false, r.key, "envelope")
	if err != nil {
		return nil
	}

	a := &ActivityEnvelope{}
	err = mdata.Cjson.Unmarshal([]byte(jsonStr), a)
	if err != nil {
		return nil
	}

	return a
}

func (r *RedEnvelopeHashRedis) GetActivityEnvelopeForMessage(c context.Context, aId, siteId int64) *ActivityEnvelope {

	jsonStr, err := core.GetKey(false, fmt.Sprintf(RedEnvelopMessageSendKey, aId, siteId))
	if err != nil {
		return nil
	}

	a := &ActivityEnvelope{}
	err = mdata.Cjson.Unmarshal([]byte(jsonStr), a)
	if err != nil {
		return nil
	}

	return a
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
