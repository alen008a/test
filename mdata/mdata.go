package mdata

import (
	"time"

	"github.com/RussellLuo/timingwheel"
	jsoniter "github.com/json-iterator/go"
)

func init() {
	TimingWheel.Start()
}

var (
	Cjson = jsoniter.ConfigCompatibleWithStandardLibrary
)

func MustMarshal(v interface{}) []byte {
	if v != nil {
		b, _ := Cjson.Marshal(v)
		return b
	}

	return []byte{}
}

func MustMarshal2String(v interface{}) string {
	if v != nil {
		b, _ := Cjson.MarshalToString(v)
		return b
	}

	return ""
}

var TimingWheel = timingwheel.NewTimingWheel(time.Millisecond, 20)

type RotateScheduler struct {
	Interval time.Duration
}

func (s *RotateScheduler) Next(prev time.Time) time.Time {
	return prev.Add(s.Interval)
}

type MsgId = uint32

const (
	MsgIdTypeHeartbeat       MsgId = 0     //心跳
	MsgIdTypeConnect               = 10000 //建立连接
	MsgIDTypeLogin                 = 10001 // 用户登陆
	MsgIDTypeJoinRoom              = 10002 // 加入房间
	MsgIDTypeBroadcastRoom         = 10003 // 在房间聊天
	MsgIDTypeSpeechStatus          = 10004 // 单播会员禁言
	MsgIDTypeMsgShield             = 10005 // 房间广播消息屏蔽
	MsgIDTypeRoomStatus            = 10006 // 单房间状态变更
	MsgIDTypeChatVIPLevel          = 10007 // 全局广播可参与VIP聊天等级
	MsgIDTypeBroadcastJoin         = 10008 // 加入房间时广播（包括是否开启入房特效，等等）
	MsgIDTypeSelfJoin              = 10016 // 加入房间时单播(用于统计活跃人数)
	MsgIDTypeAllRoomMaintain       = 10009 // 全局广播房间状态更新
	MsgIDTypeShareBetRecord        = 10010 // 分享注单
	MsgIDTypeChatHistory           = 10011 // 历史聊天记录
	MsgIDTypeLeaveRoom             = 10012 // 离开房间
	MsgIDMatchTerminated           = 10013 // 房间销毁
	MsgIDLiveScorePush             = 10014 // 房间对应的比赛比分推送
	MsgIDMatchChatClear            = 10015 // 清除聊天室
	MsgIDLiveGiftPush              = 10017 // 礼物赠送记录推送
	MsgScorePredictionPush         = 10018 // 比分预测记录推送
	MsgIDTypeNoLogin               = 10019 // 免用户登陆
	MsgIDTypeApiApp                = 10020 // WS 推送加密后的 API 下发 APP 的文件内容

	MsgIdRedPackageRain    = 40001 // 红包雨的消息逻辑
	MsgIdRedPackageReceive = 40002 // 主动接收红包消息的ID
	MsgSimpleMsgMark       = 20000

	MsgIdFinanceNotifying = 20001 //财务推送数据

	MsgBetsStatusNotifying = 30001 //推单业务推送

	MsgIDInternalNotice = 60000 //内部通知
)

const (
	HeaderToken = "X-API-TOKEN"
	HeaderTrace = "X-API-TRACE"
	HeaderSite  = "X-API-SITE" // 平台标志
	HeaderXXX   = "X-API-XXX"
)

const (
	WEB                = "web"
	AGENT_WEB          = "agent_web"
	H5                 = "h5"
	Android            = "android"
	IOS                = "ios"
	PC                 = "pc"
	SPORT_IOS          = "sport_ios"
	SPORT_Android      = "sport_android"
	AGENT_IOS          = "agent_ios"
	AGENT_Android      = "agent_android"
	CHESS_IOS          = "chess_ios"
	CHESS_Android      = "chess_android"
	LOTTERY_IOS        = "lottery_ios"
	LOTTERY_Android    = "lottery_android"
	PERSON_IOS         = "person_ios"
	PERSON_Android     = "person_android"
	ALL_SPORT_IOS      = "all_sport_ios"
	ALL_SPORT_Android  = "all_sport_android"
	GAME_IOS           = "game_ios"
	GAME_Android       = "game_android"
	DALI_CHINA_IOS     = "dali_china_ios"
	DALI_CHINA_Android = "dali_china_android"
)

// IPLoc ip信息
type IPLoc struct {
	Country  string // 国家
	Province string // 省份
	City     string // 城市
}

// CommonMapConfigRedis 后台配置公用读取key的结构体
type CommonMapConfigRedis struct {
	CommonInfoKey string `json:"commonInfoKey"`
	CreatedAt     string `json:"createdAt"`
	GroupID       int    `json:"groupId"`
	GroupValue    string `json:"groupValue"`
	ID            int    `json:"id"`
	IsDelete      int    `json:"isDelete"`
	Name          string `json:"name"`
	Remark        string `json:"remark"`
	ResourceID    string `json:"resourceId"`
	UpdatedAt     string `json:"updatedAt"`
	UpdatedBy     string `json:"updatedBy"`
	Value         string `json:"value"`
}

// PlatformStrType 平台映射map
var PlatformStrType = map[string]string{
	"ios":                "0",  // 全站ios
	"android":            "0",  // 全站android
	"h5":                 "3",  // 全站h5
	"web":                "2",  // 全站web
	"sport_ios":          "1",  // 体育ios
	"sport_android":      "1",  // 体育android
	"agent_ios":          "-1", // 代理ios
	"agent_android":      "-1", // 代理android
	"chess_ios":          "7",  // 棋牌ios
	"chess_android":      "7",  // 棋牌android
	"lottery_ios":        "6",  // 彩票ios
	"lottery_android":    "6",  // 彩票android
	"person_ios":         "8",  // 真人ios
	"all_sport_ios":      "9",  // 全站体育ios
	"all_sport_android":  "9",  // 全站android
	"game_ios":           "4",  // 电竞ios
	"game_android":       "4",  // 电竞android
	"dali_china_ios":     "11", // 达利中国ios
	"dali_china_android": "11", // 达利中国andorid

}

var APPPlatformStrType = map[string]string{
	"ios":               "0", // 全站ios
	"android":           "0", // 全站android
	"sport_ios":         "1", // 体育ios
	"sport_android":     "1", // 体育android
	"chess_ios":         "7", // 棋牌ios
	"chess_android":     "7", // 棋牌android
	"lottery_ios":       "6", // 彩票ios
	"lottery_android":   "6", // 彩票android
	"person_ios":        "8", // 真人ios
	"all_sport_ios":     "9", // 全站体育ios
	"all_sport_android": "9", // 全站android
	"game_ios":          "4", // 电竞ios
	"game_android":      "4", // 电竞android
}

type YSeriesJumpInfo struct {
	Status bool   `json:"status"` // 开关 bool  true 开 false 关
	Title  string `json:"title"`
	Url    string `json:"url"` // 跳转url
}

type AliIpJson struct {
	Ret  int    `json:"ret"`
	Msg  string `json:"msg"`
	Data struct {
		IP        string `json:"ip"`
		LongIP    string `json:"long_ip"`
		Isp       string `json:"isp"`
		Area      string `json:"area"`
		RegionID  string `json:"region_id"`
		Region    string `json:"region"`
		CityID    string `json:"city_id"`
		City      string `json:"city"`
		CountryID string `json:"country_id"`
		Country   string `json:"country"`
	} `json:"data"`
	LogID string `json:"log_id"`
}

type AliIpJsonV2 struct {
	Ret  int `json:"ret"`
	Data struct {
		Country       string `json:"country"`
		CountryCode   string `json:"country_code"`
		Prov          string `json:"prov"`
		City          string `json:"city"`
		CityCode      string `json:"city_code"`
		CityShortCode string `json:"city_short_code"`
		Area          string `json:"area"`
		PostCode      string `json:"post_code"`
		AreaCode      string `json:"area_code"`
		Isp           string `json:"isp"`
		Lng           string `json:"lng"`
		Lat           string `json:"lat"`
		LongIP        string `json:"long_ip"`
	} `json:"data"`
	Qt float64 `json:"qt"`
}
