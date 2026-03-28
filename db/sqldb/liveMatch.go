package sqldb

import (
	"msgPushSite/mdata"
)

func GetMatchDataByMatchID(siteId, matchID string) (*mdata.LiveMatch, error) {
	var (
		err error
		res = new(mdata.LiveMatch)
	)

	tx := LiveSlave().Select(`l.*,lr.site_id,lr.status,lr.enter,lr.active`).
		Table("live_match as l").Joins(" left join live_match_info_relation as lr on l.match_id = lr.match_id and lr.site_id = ?", siteId)
	err = tx.Where("l.match_id = ?", matchID).First(res).Error
	return res, err
}
