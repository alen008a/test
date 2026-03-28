package es

import (
	"bytes"
	"errors"
	"fmt"
	"msgPushSite/config"
	"msgPushSite/internal/glog"
	"msgPushSite/lib/httpclient"
	"msgPushSite/mdata"
	"msgPushSite/utils"
	"strings"
	"time"
)

const (
	ESIndexPrefix = "chat_msg_%s_"

	ConversationRecord = `
	{
		"from":%v,
		"size":%v,
        "_source":[
             "id",
             "siteId",
             "status",
             "seq",
             "vip",
             "nickname",
             "msg",
             "created_at",
             "timestamp",
             "memberId",
             "category",
             "categoryType",
             "matchId",
             "name"
        ],
		"query":{"bool" : {
			"must" : [
			 	%v
			],
            "must_not":[
                %v
            ]
			}
		},
		"sort":[
			%s
		]
	}
	`
)

type EsResp struct {
	Status      float64     `json:"status"`
	Error       interface{} `json:"error"`
	RecordToTal RecordTotal `json:"aggregations"`
	Hits        HitData     `json:"hits"`
}
type RecordTotal struct {
	CountID TotalV `json:"countId"`
}

type TotalV struct {
	Value int `json:"value"`
}

type HitData struct {
	Total HitsTotal   `json:"total"`
	Hits  []RecordHit `json:"hits"`
}

type HitsTotal struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"`
}

type RecordHit struct {
	Index  string          `json:"_index"`
	Type   string          `json:"_type"`
	ID     string          `json:"_id"`
	Source FinalRecordData `json:"_source"`
}

type FinalRecordData struct {
	ID           int64  `json:"id"`
	SiteId       int    `json:"siteId"`
	Status       int    `json:"status"`
	Seq          string `json:"seq"`          // 消息ID
	VIP          int    `json:"vip"`          // VIP等级
	Nickname     string `json:"nickname"`     // nickname
	Msg          string `json:"msg"`          // 广播内容
	CreatedAt    string `json:"created_at"`   // 消息到达时间
	Timestamp    string `json:"timestamp"`    // 消息到达时间
	MemberId     int    `json:"memberId"`     // 用户ID
	Category     int    `json:"category"`     // 消息类型
	CategoryType int    `json:"categoryType"` // 消息类型
	MatchId      string `json:"matchId"`      // 房间号
	Name         string `json:"name"`         // 发送消息的会员账号
}

func GetHistoryTimestampBySeqFromES(req *mdata.HistoryRecordReqSchema) (string, error) {
	var (
		res string
		err error
	)

	if len(req.Seq) < 1 {
		return "", err
	}
	index := fmt.Sprintf(ESIndexPrefix, req.SiteId) + time.Now().Format("2006_01")
	var (
		mustBuffer bytes.Buffer
	)
	mustBuffer.WriteString(fmt.Sprintf("{\"term\":{\"siteId\":%v}},", req.SiteId))
	mustBuffer.WriteString(fmt.Sprintf("{\"term\":{\"status\":%v}},", 1))
	mustBuffer.WriteString(fmt.Sprintf("{\"term\":{\"seq.keyword\":\"%v\"}},", strings.Trim(req.Seq, "")))

	mustStr := strings.Trim(mustBuffer.String(), ",")

	var (
		sortBuffer bytes.Buffer
	)
	sortBuffer.WriteString("{\n\"created_at\":{\n \"order\":\"desc\"\n }\n}")
	jsonStr := fmt.Sprintf(ConversationRecord, 0, 1, mustStr, "", sortBuffer.String())

	resp := EsResp{}
	dataStr := GetChatData(index, jsonStr)
	if len(dataStr) < 1 {
		return "", err
	}
	err = mdata.Cjson.Unmarshal([]byte(dataStr), &resp)
	if err != nil {
		glog.Errorf("GetHistoryIDBySeqFromES Error err=%s", err)
		return "", err
	}

	for _, v := range resp.Hits.Hits {
		return v.Source.CreatedAt, nil
	}
	return res, err
}

func GetHistoryByIDFromES(req *mdata.HistoryRecordReqSchema, lastTimeStamp string) ([]*mdata.BroadcastRoomRspSchema, int64, error) {
	var (
		res                 []*mdata.BroadcastRoomRspSchema
		err                 error
		msutNotStr, mustStr string
	)

	if len(req.BeginTime) < 1 {
		req.BeginTime = utils.GetBjNowTime().Add(-2 * time.Hour).Format(utils.TimeBarFormat)
	}
	if len(req.EndTime) < 1 {
		req.EndTime = utils.GetBjNowTime().Format(utils.TimeBarFormat)
	}
	startAt := fmt.Sprintf("%s+08:00", strings.Replace(req.BeginTime, " ", "T", -1))
	endAt := fmt.Sprintf("%s+08:00", strings.Replace(req.EndTime, " ", "T", -1))

	if lastTimeStamp != "" {
		endAtTime, err := utils.BjTBarFmtTimeFormat(lastTimeStamp, utils.TimeTBjFormat)
		if err != nil {
			startAt = utils.GetBjNowTime().Add(-2 * 24 * time.Hour).Format(utils.TimeTBjFormat)
		} else {
			startAt = endAtTime.Add(-4 * time.Hour).Format(utils.TimeTBjFormat)
		}
		endAt = lastTimeStamp
	}
	if req.Seq != "" {
		msutNotStr = fmt.Sprintf("{\"term\":{\"seq.keyword\":\"%v\"}}", req.Seq)
	}
	var (
		mustBuffer bytes.Buffer
	)
	mustBuffer.WriteString(fmt.Sprintf("{\"term\":{\"siteId\":%v}},", req.SiteId))
	mustBuffer.WriteString(fmt.Sprintf("{\"term\":{\"status\":%v}},", 1))
	mustBuffer.WriteString(fmt.Sprintf("{\"term\":{\"matchId.keyword\":\"%v\"}},", strings.Trim(req.RID, "")))
	mustBuffer.WriteString(fmt.Sprintf("{\"term\":{\"flag\":%v}},", 0))
	mustBuffer.WriteString(fmt.Sprintf("{\n\"range\":{\n\"created_at\":{\n\"from\":\"%s\",\n\"to\":\"%s\"\n}\n}\n},", startAt, endAt))
	if req.ChatCategory != 0 {
		mustBuffer.WriteString(fmt.Sprintf("{\"term\":{\"category\":%v}},", req.ChatCategory))
		if req.CategoryType != 0 {
			mustBuffer.WriteString(fmt.Sprintf("{\"term\":{\"categoryType\":%v}},", req.CategoryType))
		}
	}
	mustStr = strings.Trim(mustBuffer.String(), ",")

	from := (req.PageNum - 1) * req.PageSize
	var (
		sortBuffer bytes.Buffer
	)
	sortBuffer.WriteString("{\n\"created_at\":{\n \"order\":\"desc\"\n }\n}")
	jsonStr := fmt.Sprintf(ConversationRecord, from, req.PageSize, mustStr, msutNotStr, sortBuffer.String())
	index := GetEsIndex(fmt.Sprintf(ESIndexPrefix, req.SiteId), req.BeginTime, req.EndTime)
	resp := EsResp{}
	dataStr := GetChatData(index, jsonStr)
	if len(dataStr) < 1 {
		return res, 0, err
	}
	err = mdata.Cjson.Unmarshal([]byte(dataStr), &resp)
	if err != nil {
		glog.Errorf("GetHistoryIDBySeqFromES Error err=%s", err)
		return res, 0, err
	}

	cnt := len(resp.Hits.Hits)
	for i := 1; i <= cnt; i++ {
		tmp := &mdata.BroadcastRoomRspSchema{
			SiteId:       resp.Hits.Hits[cnt-i].Source.SiteId,
			Seq:          resp.Hits.Hits[cnt-i].Source.Seq,
			VIP:          resp.Hits.Hits[cnt-i].Source.VIP,
			Nickname:     resp.Hits.Hits[cnt-i].Source.Nickname,
			Msg:          resp.Hits.Hits[cnt-i].Source.Msg,
			Timestamp:    resp.Hits.Hits[cnt-i].Source.Timestamp,
			MemberId:     resp.Hits.Hits[cnt-i].Source.MemberId,
			Category:     resp.Hits.Hits[cnt-i].Source.Category,
			CategoryType: resp.Hits.Hits[cnt-i].Source.CategoryType,
		}
		res = append(res, tmp)
	}
	return res, int64(resp.Hits.Total.Value), nil
}

func GetChatData(index, JsonStr string) string {
	postUrl := fmt.Sprintf("http://%s:%s@%s/%s/_search", config.GetChatESConfig().AuthUser, config.GetChatESConfig().AuthPass, config.GetChatESConfig().Address, index)
	header := map[string]string{}
	res, err := httpclient.POSTJson(postUrl, []byte(JsonStr), header, nil)
	if err != nil {
		glog.Errorf("EsSearchErr=%v", err)
		return ""
	}
	glog.Infof("ES GetChatData|index=>%v|result=>%v|params=>%v", index, string(res), JsonStr)
	return string(res)
}

func GetEsIndex(esIndexName string, startAt string, endAt string) (index string) {
	// 开始 结束时间不超过180天
	loc, _ := time.LoadLocation("Asia/Shanghai")
	startDate, _ := time.ParseInLocation("2006-01-02 15:04:05", startAt, loc)
	endDate, _ := time.ParseInLocation("2006-01-02 15:04:05", endAt, time.Local)
	startDate = AddDate(startDate, 0, -2, 0)
	endDate = AddDate(endDate, 0, 1, 0)
	// 如果结束日期是大于当前日期，当前日期作为结束日期
	if time.Now().UTC().After(endDate) {
		endDate = time.Now()
		endAt = endDate.Format("2006-01-02 15:04:05")
	}
	monthDiff := SubMonth(endDate, startDate) + 1
	indexStr := ""
	for i := 0; i < monthDiff; i++ {
		var date = AddDate(startDate, 0, i, 0)
		tempIndex := esIndexName + date.Format("2006_01")
		if checkIndexExist(tempIndex) {
			if len(indexStr) > 0 {
				indexStr = indexStr + ","
			}
			indexStr = indexStr + tempIndex
		}
	}
	return indexStr
}

// 计算两个日期相差月份数
func SubMonth(t1, t2 time.Time) (month int) {
	y1 := t1.Year()
	y2 := t2.Year()
	//获取月
	m1 := int(t1.Month())
	m2 := int(t2.Month())

	//获取相差年数
	yearInterval := y1 - y2
	//如果t1的月份小于t2的月份，或者两者的月份相等但t1的天数小于t2的天数，则将相差年减1
	if m1 < m2 || (m1 == m2 && y1 == y2 && yearInterval > 0) {
		yearInterval--
	}
	// 获取月数差值
	monthInterval := (m1 + 12) - m2
	monthInterval %= 12
	month = yearInterval*12 + monthInterval
	return
}

func AddDate(t time.Time, year, month, day int) time.Time {
	targetDate := t.AddDate(year, month, -t.Day()+1)
	targetDay := targetDate.AddDate(0, 1, -1).Day()
	if targetDay > t.Day() {
		targetDay = t.Day()
	}
	targetDate = targetDate.AddDate(0, 0, targetDay-1+day)
	return targetDate
}

// 查询索引是否存在
func checkIndexExist(index string) bool {
	basicAuth := httpclient.BasicAuth{}
	basicAuth.Username = config.GetChatESConfig().AuthUser
	basicAuth.Password = config.GetChatESConfig().AuthPass
	headerMap := make(map[string]string)
	headerMap["content-type"] = "application/json"
	path := fmt.Sprintf("http://%s:%s@%s/%s", config.GetChatESConfig().AuthUser, config.GetChatESConfig().AuthPass, config.GetChatESConfig().Address, index)
	statusCode, err := httpclient.CheckESIndexesExists(path, headerMap, basicAuth)
	if err == nil && statusCode == 200 {
		return true
	}
	return false
}

// 以seq获取聊天数据
func GetMsgDataBySeqFromES(req *mdata.HistoryRecordReqSchema) (*mdata.LiveMatchMessage, error) {

	var res = new(mdata.LiveMatchMessage)
	var err error

	if len(req.Seq) < 1 {
		return nil, errors.New("invalid seq")
	}

	index := fmt.Sprintf("%v*", fmt.Sprintf(ESIndexPrefix, req.SiteId))
	var (
		mustBuffer bytes.Buffer
	)
	mustBuffer.WriteString(fmt.Sprintf("{\"term\":{\"seq.keyword\":\"%v\"}},", strings.Trim(req.Seq, "")))

	mustStr := strings.Trim(mustBuffer.String(), ",")

	from := 0
	limit := 1
	var (
		sortBuffer bytes.Buffer
	)
	sortBuffer.WriteString("{\n\"created_at\":{\n \"order\":\"desc\"\n }\n}")
	jsonStr := fmt.Sprintf(ConversationRecord, from, limit, mustStr, "", sortBuffer.String())

	resp := EsResp{}
	dataStr := GetChatData(index, jsonStr)
	if len(dataStr) < 1 {
		return res, err
	}
	err = mdata.Cjson.Unmarshal([]byte(dataStr), &resp)
	if err != nil {
		glog.Errorf("GetHistoryIDBySeqFromES Error err=%s", err)
		return res, err
	}
	if len(resp.Hits.Hits) > 0 {
		tmp := resp.Hits.Hits[0].Source
		res.Msg = tmp.Msg
		res.Seq = tmp.Seq
		res.Nickname = tmp.Nickname
		res.Category = tmp.Category
		res.CategoryType = tmp.CategoryType
		res.Vip = tmp.VIP
		res.Timestamp = tmp.Timestamp
		res.MemberId = tmp.MemberId
		res.MatchId = tmp.MatchId
		res.Name = tmp.Name
	}
	return res, nil
}
