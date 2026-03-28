package mdata

const (
	LiveMatchMsgReportTable   = "live_match_message_report"
	DefaultWarnCount          = 3                            //默认被举报达到推送预警的次数
	DefaultMaxReportCount     = 10                           //默认每个会员每天允许的最大举报次数
	MaxReportCountRedisKey    = "LiveMsgMaxReportCount_%s"   //每个会员每天允许的最大举报次数redis key
	TriggerCountRedisKey      = "LiveMsgMaxTriggerCount_%s"  //触发预警的举报次数redis key
	MemberReportCountRedisKey = "MemberReportCount_%s_%s_%s" //会员每天举报次数redis key
	ChatMsgWarnFlagRedisKey   = "LiveMsgWarnFlag_%s_%s"      //某条消息被举报的预警推送标识 redis key
	ChatMsgReportedRedisKey   = "LiveMsgReported_%s_%s"      //某条消息的举报会员名单集合 redis key

	ChatMsgReportTemplate = `
		*****聊天室消息举报预警*****
        - 站点 : %s 
		- 平台 : %s
		- 场馆名称 : %s
		- 对阵双方 : %s
		- 举报人账号 : %s
		- 被举被人账号及VIP等级 : %s | %s
		- 被举报的消息内容 : %s
		- 发言时间 : %s
	`
)

// 举报原因配置
var ReportReasons = []map[string]interface{}{
	{
		"code":   "1",
		"reason": "涉嫌广告",
	},
	{
		"code":   "2",
		"reason": "恶意刷屏",
	},
	{
		"code":   "3",
		"reason": "脏话谩骂",
	},
	{
		"code":   "4",
		"reason": "谈论政治",
	},
	{
		"code":   "5",
		"reason": "其他",
	},
}

// 前台举报消息提交的参数
type LiveMatchMsgReportParams struct {
	Token  string `json:"token"`  //对应前端 用户前台 X-API-TOKEN
	Body   string `json:"body"`   //JWToken
	Seq    string `json:"seq"`    //消息唯一标识(未避免以后消息举报表数据写es, 这里不用id)
	Reason string `json:"reason"` //举报原因
}

// 聊天室消息举报记录表结构
type LiveMatchMessageReport struct {
	Id           int64  `gorm:"column:id" json:"id"`
	SiteId       int    `gorm:"column:site_id" json:"siteId"`                //站点ID
	Msg          string `gorm:"column:msg" json:"msg"`                       //消息内容
	Seq          string `gorm:"column:seq" json:"seq"`                       //对应消息表中seq字段
	Name         string `gorm:"column:name" json:"name"`                     //被举报人账号
	Vip          int    `gorm:"column:vip" json:"vip"`                       //被举报人账号等级
	ReporterName string `gorm:"column:reporter_name" json:"reporter_name"`   //举报人账号
	ReporterVip  int    `gorm:"column:reporter_vip" json:"reporter_vip"`     //举报人账号等级
	Category     int    `gorm:"column:category" json:"category"`             //消息类型 1用户消息 2晒单
	MatchId      string `gorm:"column:match_id" json:"match_id"`             //赛事id
	Venue        string `gorm:"column:venue" json:"venue"`                   //场馆中文名称
	League       string `gorm:"column:league" json:"league"`                 //联赛名称
	Home         string `gorm:"column:home" json:"home"`                     //主队名称
	Away         string `gorm:"column:away" json:"away"`                     //客队名称
	ReportReason string `gorm:"column:report_reason" json:"report_reason"`   //消息被举报原因
	MsgCreatedAt string `gorm:"column:msg_created_at" json:"msg_created_at"` //被举报的消息创建时间
	CreatedAt    string `gorm:"column:created_at" json:"created_at"`         //举报时间
	UpdatedAt    string `gorm:"column:updated_at" json:"updated_at"`         //记录修改时间
}

func GetVenueCnNameByCode(venue string) string {
	switch venue {
	case "IMTY":
		return "IM体育"
	case "YBTY":
		return "OB体育"
	case "XJTY":
		return "小金体育"
	case "FBTY":
		return "FB体育"
	}
	return venue
}

func CheckMsgReportReason(code string) bool {
	for _, v := range ReportReasons {
		if code == v["code"] {
			return true
		}
	}
	return false
}
