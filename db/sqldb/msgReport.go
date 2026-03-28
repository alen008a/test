package sqldb

import (
	"errors"
	"fmt"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/mdata"
)

// CheckMsgReportedOrNot 检查消息是否被某个会员举报过
func CheckMsgReportedOrNot(siteId, seq, username string) (bool, error) {
	if len(seq) < 1 || len(username) < 1 {
		return false, errors.New("invalid seq or username")
	}

	//先查redis, redis没有再查tidb
	reported, err := core.SIsMember(fmt.Sprintf(mdata.ChatMsgReportedRedisKey, siteId, seq), username)
	if err == nil {
		return reported, nil
	}

	var count int64
	err = LiveSlave().Table(mdata.LiveMatchMsgReportTable).
		Where("site_id = ? and seq = ? and reporter_name = ?", siteId, seq, username).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func CreateMsgReportData(data *mdata.LiveMatchMessageReport) error {
	return Live().Table(mdata.LiveMatchMsgReportTable).Create(&data).Error
}
