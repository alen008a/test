package mdata

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	SecretKey                 = "ga8e31f.53a49f648f489ds"
	NotJoinRoomErr            = errors.New("请先加入房间")
	UserSpeechErr             = errors.New("当前已被禁言")
	UserVIPLevelErr           = errors.New("未达到发言要求")
	RoomStatusErr             = errors.New("聊天室维护中")
	GlobalRoomStatusErr       = errors.New("聊天室维护中")
	RoomNotStartErr           = errors.New("赛事还未开始")
	SerViceStatusErr          = errors.New("当前服务端异常")
	TokenParserErr            = errors.New("登录未通过")
	TokenExpireErr            = errors.New("token过期")
	TokenInvalidErr           = errors.New("token不可用")
	ArgsParserErr             = errors.New("传递参数出错")
	UserNotLoginErr           = errors.New("请先进行登录")
	RoomNotFoundErr           = errors.New("房间不存在")
	AllRoomMaintainErr        = errors.New("聊天室正在维护")
	ServiceNotInitErr         = errors.New("聊天室未打开，请联系客服")
	MsgBodyEmptyErr           = errors.New("消息文本为空")
	RepeatLoginErr            = errors.New("您已经登录过了，请勿重复校验")
	RepeatJoinRoomErr         = errors.New("您当前已在所在房间，请勿重复加入")
	JoinFrequencyErr          = errors.New("加入房间太过于频繁，请稍后再试")
	SpeechFrequencyErr        = errors.New("为保证聊天室的良好环境，请不要过于频繁发言，每3分钟最多发送一条消息。")
	SpeechFrequencyNormalErr  = errors.New("为保证聊天室的良好环境，请不要过于频繁发言")
	ShareBetAmountLimitErr    = errors.New("请分享大于等于100元的注单")
	ShareRecordRepeatErr      = errors.New("请勿重复多次晒单")
	MsgLengthLimitExceededErr = errors.New("发送聊天内容长度大于150字符")
	FrequentOperationLimitErr = errors.New("操作过于频繁，请稍后重试")
	SensitiveRepeatedError    = errors.New("重复字符或者数字敏感词")
)

type MemberInfo struct {
	Id       int    `gorm:"column:id" json:"id"`              // 自增主键id
	SiteId   string `gorm:"column:site_id" json:"siteId"`     // 站点ID
	Name     string `gorm:"column:name" json:"name"`          // 用户名（登录账号）
	NickName string `gorm:"column:nick_name" json:"nickName"` // 昵称
	Vip      int    `json:"vip"`                              // VIP等级
	UUID     string `json:"uuid"`                             // 设备号
	IsAgent  string `json:"isAgent"`                          // 1:是代理
	CreateAt string `json:"createAt"`                         // 账号创建时间
	*jwt.StandardClaims
}

func ParserToken(req *LoginReqSchema) (*MemberInfo, error) {
	token, err := jwt.ParseWithClaims(req.Body, &MemberInfo{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey + req.Token), nil
	})

	if err != nil {
		var ve *jwt.ValidationError
		if errors.As(err, &ve) {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, TokenInvalidErr
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, TokenExpireErr
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, TokenInvalidErr
			} else {
				return nil, TokenInvalidErr
			}
		}
	}
	if claims, ok := token.Claims.(*MemberInfo); ok && token.Valid {
		return claims, nil
	} else {
		return nil, err
	}
}

func GenerateToken(id, vip int, name, token, createAt string) (string, error) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &MemberInfo{
		Id:       id,
		Name:     name,
		Vip:      vip,
		CreateAt: createAt,
		StandardClaims: &jwt.StandardClaims{
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		},
	}).SignedString([]byte(SecretKey + token))

	if err != nil {
		return "", err
	}
	return token, err
}

type HistoryRecordReqSchema struct {
	SiteId       string `json:"siteId"`
	Seq          string `json:"seq"`
	RID          string `json:"rid"`
	PageNum      int    `json:"pageNum"`
	PageSize     int    `json:"pageSize"`
	ChatCategory int    `json:"category"`     //0表所有 1表聊天 2表晒单
	CategoryType int    `json:"categoryType"` //0 所有 如果 category晒单 ，type -1为普通单 2 为大单
	BeginTime    string `json:"beginTime"`    //查询起始时间
	EndTime      string `json:"endTime"`      //查询截止时间
}

type HistoryRecordReqHTTPSchema struct {
	Token        string `json:"token" binding:"required"` //对应前端 用户前台 X-API-TOKEN
	Body         string `json:"body" binding:"required"`  //JWToken
	Seq          string `json:"seq"`
	RID          string `json:"rid" binding:"required"`
	PageNum      int    `json:"pageNum" binding:"required,gte=0"`
	PageSize     int    `json:"pageSize" binding:"required,gte=0"`
	ChatCategory int    `json:"category"`     //0表所有 1表聊天 2表晒单
	CategoryType int    `json:"categoryType"` //0 所有 如果 category晒单 ，type -1为普通单 2 为大单
	BeginTime    string `json:"beginTime"`    //查询起始时间
	EndTime      string `json:"endTime"`      //查询截止时间
}

// LoginReqSchema 登陆传参
type LoginReqSchema struct {
	Token string `json:"token"` //对应前端 用户前台 X-API-TOKEN
	Body  string `json:"body"`  //JWToken
	RID   string `json:"rid"`   //房间ID
}

// LoginRspSchema 登陆响应
type LoginRspSchema struct {
	SpeechStatus int                       `json:"speechStatus"`         // 会员是否被禁言 0 正常 1 被禁言
	VipMin       int                       `json:"vipMin"`               // 发言最低等级
	VipMax       int                       `json:"vipMax"`               // 发言最高等级
	EffectMin    int                       `json:"effectMin"`            // 特效vip最低等级
	EffectMax    int                       `json:"effectMax"`            // 特效vip最高等级
	LiveDate     string                    `json:"liveDate,omitempty"`   // 赛事开始时间（当登陆传roomID时）
	LiveStatus   int                       `json:"roomStatus,omitempty"` // 房间状态 1进行中 2停用（当登陆传roomID时）
	History      []*BroadcastRoomRspSchema `json:"history,omitempty"`    // 历史聊天记录（当登陆传roomID时)
	EffectOpen   int                       `json:"effectOpen"`           // 特效是否开启 0 是 1 否
	BsOpen       bool                      `json:"bsOpen"`               // 红单推荐入口
	AdvOpen      bool                      `json:"advOpen"`              //返回广告banner 开关状态
	GiftOpen     bool                      `json:"giftOpen"`             // 礼物打赏入口
	CopyOpen     bool                      `json:"copyOpen"`             // 是否能复制内容开关
	ScoreOpen    bool                      `json:"scoreOpen"`            // 比分预测开关
	Notify       string                    `json:"speechNotify"`         // 提示语 可以为空
	IpServer     []*IpServer               `json:"ipServer"`             // ip服务地址
	IpServerApp  []string                  `json:"ipServerApp"`          // WS 推送加密后的 API 下发 APP 的文件内容
}

// JoinRoomReqSchema 进入房间传参
type JoinRoomReqSchema struct {
	Rid string `json:"rid"`
}

// UnAuthJoinRoomRspSchema 未登录用户返回内容
type UnAuthJoinRoomRspSchema struct {
	LiveDate     string                    `json:"liveDate"`     // 赛事开始时间
	LiveStatus   int                       `json:"roomStatus"`   // 房间状态 1进行中 2停用
	History      []*BroadcastRoomRspSchema `json:"history"`      // 历史聊天记录
	Next         bool                      `json:"next"`         // 是否存在历史消息（客户端翻页用）
	BulletOpen   bool                      `json:"bulletOpen"`   //是否默认开启弹幕
	BulletButton bool                      `json:"bulletButton"` // 弹幕 开关是否 展示
	EffectOpen   int                       `json:"effectOpen"`   // 特效是否开启 0 是 1 否
	BsOpen       bool                      `json:"bsOpen"`       // 红单推荐入口
	AdvOpen      bool                      `json:"advOpen"`      //返回广告banner 开关状态
	GiftOpen     bool                      `json:"giftOpen"`     // 礼物打赏入口
	CopyOpen     bool                      `json:"copyOpen"`     // 是否能复制内容开关
	ScoreOpen    bool                      `json:"scoreOpen"`    // 比分预测开关
}

// AuthJoinRoomBroadcastRspSchema 进入房间广播
type AuthJoinRoomBroadcastRspSchema struct {
	Msg          string `json:"msg"`          // 消息
	Nickname     string `json:"nickname"`     // 用户昵称
	VIP          int    `json:"vip"`          // VIP等级
	EffectStatus int    `json:"effectStatus"` // 特效是否开启 0-否 1-是
	Category     int    `json:"category"`     // 3为加入房间消息
	MemberId     int    `json:"memberId"`     //会员id
}

// AuthJoinRoomSelfRspSchema 进入房间单播
type AuthJoinRoomSelfRspSchema struct {
	LiveDate     string                    `json:"liveDate"`     // 赛事开始时间
	LiveStatus   int                       `json:"roomStatus"`   // 房间状态 1进行中 2停用
	History      []*BroadcastRoomRspSchema `json:"history"`      // 历史聊天记录
	Next         bool                      `json:"next"`         // 是否存在历史消息（客户端翻页用）
	BulletOpen   bool                      `json:"bulletOpen"`   // 是否默认开启弹幕
	BulletButton bool                      `json:"bulletButton"` // 弹幕 开关是否 展示
	EffectOpen   int                       `json:"effectOpen"`   // 特效是否开启 0-否 1-是
	SpeechStatus int                       `json:"speechStatus"` // 会员是否被禁言 false 已被禁言 true 正常
	Notify       string                    `json:"speechNotify"` // 提示语 可以为空
	VipMin       int                       `json:"vipMin"`       // 发言最低等级
	VipMax       int                       `json:"vipMax"`       // 发言最高等级
	EffectMin    int                       `json:"effectMin"`    // 特效vip最低等级
	EffectMax    int                       `json:"effectMax"`    // 特效vip最高等级
	BsOpen       bool                      `json:"bsOpen"`       // 红单推荐入口
	AdvOpen      bool                      `json:"advOpen"`      // 返回广告banner 开关状态
	GiftOpen     bool                      `json:"giftOpen"`     // 礼物打赏入口
	CopyOpen     bool                      `json:"copyOpen"`     // 是否能复制内容开关
	ScoreOpen    bool                      `json:"scoreOpen"`    // 比分预测开关
}

// BroadcastRoomReqSchema 聊天传参
type BroadcastRoomReqSchema struct {
	Msg string `json:"msg"` // 广播内容
}

// BroadcastRoomRspSchema 聊天传参
type BroadcastRoomRspSchema struct {
	SiteId       int    `json:"siteId"`       // 站点ID
	Seq          string `json:"seq"`          // 消息ID
	VIP          int    `json:"vip"`          // VIP等级
	Nickname     string `json:"nickname"`     // nickname
	Msg          string `json:"msg"`          // 广播内容
	Timestamp    string `json:"timestamp"`    // 消息到达时间
	MemberId     int    `json:"memberId"`     // 用户ID
	Category     int    `json:"category"`     // 消息类型
	CategoryType int    `json:"categoryType"` // 消息类型
	AllowReport  int    `json:"allowReport"`  // 是否允许举报标识： 1是  0否
	IsReported   int    `json:"isReported"`   // 是否已经被当前客户端用户举报过标识： 1是  0否
}

// BroadcastRoomKafkaSchema 聊天入库Kafka
type BroadcastRoomKafkaSchema struct {
	RoomId        string `json:"roomId"`
	SiteId        int    `json:"siteId"`
	MsgId         uint32 `json:"msgId"`
	MsgFlag       uint32 `json:"msgFlag"`
	Name          string `json:"name"`
	Seq           string `json:"seq"`           // 消息ID
	Flag          int64  `json:"flag"`          // 消息标记 命中敏感词 0-没有命中 1 命中铭感词 2-命中连续字符
	SensitiveWord string `json:"sensitiveWord"` //命中哪个铭感词
	Status        int    `json:"status"`        // 推送状态 0 屏蔽, 1 发送  2 比赛结束
	VIP           int    `json:"vip"`           // VIP等级
	Nickname      string `json:"nickname"`      // nickname
	Msg           string `json:"msg"`           // 广播内容
	Timestamp     string `json:"timestamp"`     // 消息到达时间
	AdminId       int64  `json:"adminId"`
	Venue         string `json:"venue"`
	League        string `json:"league"`
	Home          string `json:"home"`
	Away          string `json:"away"`
	MemberId      int    `json:"memberId"`     // 用户ID
	Category      int    `json:"category"`     // 消息类型
	CategoryType  int    `json:"categoryType"` // 消息类型
	MatchCate     string `json:"matchCate"`
	EffectType    int64  `json:"effectType"`
	MatchId       string `json:"matchId"`
	AllowReport   int    `json:"allowReport"`           // 是否允许举报标识： 1是  0否
	IsReported    int    `json:"isReported"`            // 是否已经被当前客户端用户举报过标识： 1是  0否
	CreatedAt     string `json:"created_at,omitempty"`  // 发言时间
	UpdatedAt     string `json:"updated_at,omitempty"`  // 更新时间
	EsIndexName   string `json:"esIndexName,omitempty"` // 索引名称
}

func (b *BroadcastRoomKafkaSchema) Bytes() []byte {
	if b != nil {
		bytes, _ := Cjson.Marshal(b)
		return bytes
	}
	return []byte{}
}

type RoomStatusSchema struct {
	Id      string `json:"id"`      // 房间ID
	Status  int    `json:"status"`  // 房间状态 0 为正常开启 1为关闭
	StartAt int64  `json:"startAt"` // 开始时间
}

type PulseInfo struct {
	Server string        `json:"server"`
	Hub    int           `json:"hub"`
	Rooms  []interface{} `json:"rooms"`
}

type Room struct {
	Name      string `json:"name"`
	Clients   int    `json:"clients"`
	Created   string `json:"created"`   // 房间创建时间
	StartDate string `json:"startDate"` // 房间赛事开始时间
}

type T13Body struct {
	RoomIds []string `json:"rids"`
}

type BannedInfo struct {
	BannedStart string `json:"bannedStart"` //禁言开始时间
	Duration    int    `json:"duration"`    //禁言时长 （单位:小时）
}

type IpServer struct {
	Grade      int    `json:"grade"`
	ClientType string `json:"clientType"`
	URL        string `json:"url"`
}

// 晒单 为兼容App端数据使用interface接收,Andirod是字符串，ios是数字，改动地方比较多，暂时不动
type ShareBetRecordDO struct {
	BetResult    interface{} `json:"betResult"`
	BetList      []BetRecord `json:"betList"`
	ComboName    interface{} `json:"comboName"`
	CurrencyID   interface{} `json:"currencyId"`
	StrBetAmount interface{} `json:"strBetAmount"`
	StrBetWin    interface{} `json:"strBetWin"`
}

// 注单记录
type BetRecord struct {
	StrStartTime      interface{} `json:"strStartTime"`
	PlayID            interface{} `json:"playId"`
	SelectionType     interface{} `json:"selectionType"`
	StrBetResult      interface{} `json:"strBetResult"`
	CanBet            interface{} `json:"canBet"`
	StrBetHandcap     interface{} `json:"strBetHandcap"`
	SportType         interface{} `json:"sportType"`
	StrAwayScore      interface{} `json:"strAwayScore"`
	MarketID          interface{} `json:"marketId"`
	IsEu              interface{} `json:"isEu"`
	StrHomeScore      interface{} `json:"strHomeScore"`
	StrLGName         interface{} `json:"strLGName"`
	StrBetOdds        interface{} `json:"strBetOdds"`
	StrAwayTeam       interface{} `json:"strAwayTeam"`
	StrBetTypeName    interface{} `json:"strBetTypeName"`
	StrBetMatchResult interface{} `json:"strBetMatchResult"`
	StrBetInfo        interface{} `json:"strBetInfo"`
	StrHomeTeam       interface{} `json:"strHomeTeam"`
	SportName         interface{} `json:"sportName"`
	SectionID         interface{} `json:"sectionId"`
	EventID           interface{} `json:"eventId"`
	PeriodID          interface{} `json:"periodId"`
	MatchType         interface{} `json:"matchType"`
	RealHandcap       interface{} `json:"realHandcap"`
}

type ShareBetRecordParam struct {
	BetResult    interface{}      `json:"a"`
	BetList      []BetRecordParam `json:"b"`
	ComboName    interface{}      `json:"c"`
	CurrencyID   interface{}      `json:"d"`
	StrBetAmount interface{}      `json:"e"`
	StrBetWin    interface{}      `json:"f"`
}

type BetRecordParam struct {
	StrStartTime      interface{} `json:"s1"`
	PlayID            interface{} `json:"s2"`
	SelectionType     interface{} `json:"s3"`
	StrBetResult      interface{} `json:"s4"`
	CanBet            interface{} `json:"s5"`
	StrBetHandcap     interface{} `json:"s6"`
	SportType         interface{} `json:"s7"`
	StrAwayScore      interface{} `json:"s8"`
	MarketID          interface{} `json:"s9"`
	IsEu              interface{} `json:"s10"`
	StrHomeScore      interface{} `json:"s11"`
	StrLGName         interface{} `json:"s12"`
	StrBetOdds        interface{} `json:"s13"`
	StrAwayTeam       interface{} `json:"s14"`
	StrBetTypeName    interface{} `json:"s15"`
	StrBetMatchResult interface{} `json:"s16"`
	StrBetInfo        interface{} `json:"s17"`
	StrHomeTeam       interface{} `json:"s18"`
	SportName         interface{} `json:"s19"`
	SectionID         interface{} `json:"s20"`
	EventID           interface{} `json:"s21"`
	PeriodID          interface{} `json:"s22"`
	MatchType         interface{} `json:"s23"`
	RealHandcap       interface{} `json:"s24"`
}
