package mdata

type History struct {
	RoomID  string                    `json:"roomId"`  // 房间ID
	History []*BroadcastRoomRspSchema `json:"history"` // 历史聊天记录
}

// VIPConf VIP配置参数
type VIPConf struct {
	VipMin     int `gorm:"vip_min" json:"vipMin"`         // 可参与聊天vip最低等级
	VipMax     int `gorm:"vip_max" json:"vipMax"`         // 可参与聊天vip最高等级
	EffectMin  int `gorm:"effect_min" json:"effectMin"`   // 特效vip最低等级
	EffectMax  int `gorm:"effect_max" json:"effectMax"`   // 特效vip最高等级
	EffectOpen int `gorm:"effect_open" json:"effectOpen"` // 特效功能 0 关闭 1 开启
}

// LiveMatchMessage 聊天记录表
type LiveMatchMessage struct {
	Msg          string `gorm:"column:msg" json:"msg"`
	Seq          string `gorm:"column:seq" json:"seq"`
	Name         string `gorm:"column:name" json:"name"`
	Nickname     string `gorm:"nickname" json:"nickname"`
	Category     int    `gorm:"column:category" json:"category"`
	CategoryType int    `gorm:"column:category_type" json:"categoryType"` //1普通 2 大单
	MatchCate    string `gorm:"column:match_cate" json:"matchCate"`
	EffectType   int64  `gorm:"column:effect_type" json:"effectType"`
	MatchId      string `gorm:"column:match_id" json:"matchId"`
	MemberId     int    `gorm:"column:member_id" json:"memberId"`
	Vip          int    `gorm:"column:vip" json:"vip"`
	Flag         int64  `gorm:"column:flag" json:"flag"`
	Timestamp    string `gorm:"column:timestamp" json:"timestamp"`
	Status       int64  `gorm:"column:status" json:"status"`
	AdminId      int64  `gorm:"column:admin_id" json:"adminId"`
}

// LiveMatch 聊天室表
type LiveMatch struct {
	Id        int64  `gorm:"column:id" json:"id"`                // 自增主键id
	MatchId   string `gorm:"column:match_id" json:"matchId"`     // 赛事id 对应赛事数据eid
	LeagueId  int    `gorm:"column:league_id" json:"leagueId"`   // 联赛id 对应赛事数据cid
	League    string `gorm:"column:league" json:"league"`        // 联赛
	LeagueEn  string `gorm:"column:league_en" json:"leagueEn"`   // 联赛
	Venue     string `gorm:"column:venue" json:"venue"`          // 场馆 XJTY 小金体育,IMTY IM体育 ,YBTY ob体育
	MatchCate string `gorm:"column:match_cate" json:"matchCate"` // 赛事类型
	Live      int    `gorm:"column:live" json:"live"`            // 赛事状态 0 为开始 1进行中 2结束
	LiveDate  string `gorm:"column:live_date" json:"liveDate"`   // 开赛时间
	Home      string `gorm:"column:home" json:"home"`            // 主队
	HomeEn    string `gorm:"column:home_en" json:"homeEn"`       // 主队 英文
	Away      string `gorm:"column:away" json:"away"`            // 客队
	AwayEn    string `gorm:"column:away_en" json:"awayEn"`       // 客队 英文
	Status    int    `gorm:"column:status" json:"status"`        // 状态 启用1 停用2
	AdminId   int    `gorm:"column:admin_id" json:"adminId"`     // 操作人员id
}

// ResEnvelopeMsgVo ## 红包雨相关的处理
type ResEnvelopeMsgVo struct {
	StartTime     int64  `json:"startTime"`              //红包雨开始时间（时间戳）
	RetmainTime   int64  `json:"retmainTime"`            //红包雨持续时间（秒钟）
	RedPackId     int64  `json:"redPackId"`              //活动ID
	Session       int64  `json:"session"`                //活动场次
	CountDownTime int64  `json:"countDownTime"`          //倒计时时间（分钟）
	Style         string `json:"style"`                  //红包样式CODE
	Status        int64  `json:"status"`                 //红包雨状态（1-启动，4-停止）
	CurrentTime   int64  `json:"currentTime"`            //服务器当前时间（时间戳）
	GamePlatform  string `json:"gamePlatform"`           //游戏设备(1-全站APP，2-体育APP)
	ActivityType  int64  `json:"activityType,omitempty"` // 0 为全站 1 为个人用户组
}

type ResEnvelopReq struct {
	MsgId   int               `json:"msgId"`
	MsgFlag int               `json:"msgFlag"`
	MsgData *ResEnvelopeMsgVo `json:"msgData"`
	Seq     string            `json:"seq"`
}

type OnlineUserReq struct {
	UserName string `json:"user_name"`
	PageSize int    `json:"page_size"`
	PageNum  int    `json:"page_num"`
	Hub      int    `json:"hub"`
	SiteId   int    `json:"site_id"`
	Type     int    `json:"type"` // 1 表示 直接查询用户名称 2 表示查询指定页面的数据 3 表示查询Hub的用户名
}

type ValidTokenReq struct {
	Token string `json:"token"`
	Body  string `json:"body"` //JWToken
}

// GetUserDeviceReq 请求体结构
type GetUserDeviceReq struct {
	AppKey string `json:"appKey" binding:"required"`
	UserId string `json:"userId" binding:"required"`
	SiteId int    `json:"siteId" binding:"required"`
}

type MsgData struct {
	CountDownTime int64  `json:"countDownTime"`
	CurrentTime   int64  `json:"currentTime"`
	GamePlatform  string `json:"gamePlatform"`
	RedPackId     int64  `json:"redPackId"`
	RetmainTime   int64  `json:"retmainTime"`
	Session       int64  `json:"session"`
	SiteId        int64  `json:"siteId"`
	StartTime     int64  `json:"startTime"`
	Status        int    `json:"status"`
	Style         string `json:"style"`
	Type          int    `json:"type"`
}

type MsgObject struct {
	SiteId  string  `json:"siteId"`  // 外层 siteId，可以为空
	Seq     string  `json:"seq"`     // 序列号，全局唯一
	MsgFlag int     `json:"msgFlag"` // 消息标志
	MsgId   int     `json:"msgId"`   // 消息 ID
	MsgData MsgData `json:"msgData"` // 业务参数
}

type MsgRequest struct {
	Count int       `json:"count"`
	Msg   MsgObject `json:"msg"`
	Delay int       `json:"delay"`
}

type OnlineUserRes struct {
	UserName string      `json:"user_name"`
	SiteId   string      `json:"site_id"`
	Hub      int         `json:"hub"`
	Info     *MemberInfo `json:"info"`
}
