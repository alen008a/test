package rediskey

const (
	ClientOnlineSetPrefix       = "cli_online_set_*"
	ClientOnlineSet             = "cli_online_set_%s" //${siteId}
	ClientOnlineClientSetPrefix = "cli_online_type_set_*"
	ClientOnlineClientSet       = "cli_online_type_set_%s_%s"     //${siteId}${client}
	ClientSendCount             = "cli_send_count_%s_%s"          //${siteId}${seq}
	ClientNoticeSet             = "client_notice_set_%s_%s_%s"    //${siteId}${name}${client}
	ClientNoticeKey             = "his_%s_%s_%s"                  //${siteId}${seq}${name}
	ClientNoticeRecordHash      = "client_notice_record_%s_%s_%s" //${siteId}${trance}${client}

	LoginParseIpRecords = "site_login_parse_ip_records_%s"

	RainKey                     = "game_facade_delay_red_envelopRain_lock_getBalance_%d"
	DelayRainKey                = "game_facade_red_envelopRain_lock_getBalance_%d"
	MarqueeKey                  = "activity_red_envelope_aid_%d_sid_%d_marquee"
	MembersDrawKey              = "activity_red_envelope_aid_%d_sid_%d_members"
	MembersLimitKey             = "activity_red_envelope_aid_%d_sid_%d_member_red_envelope_limit"
	MemberTagKey                = "activity_red_envelope_aid_%d_sid_%d_member_tag"
	RedEnvelopeKey              = "activity_red_envelope_aid_%d_sid_%d_object"
	RedEnvelopRecordKey         = "activity_red_envelope_aid_%d_sid_%d_records"
	RedEnvelopSessionRecordsKey = "activity_red_envelope_aid_%d_sid_%d_session_%d_records_%d"
	RedEnvelopeStartKey         = "activity_red_envelope_start_%d_site"

	HasRedEnvelopeSessionKey      = "has_red_envelope_session_%d_site" // 查看是否有全站的红包雨
	CurrentRedEnvelopeActivityKey = "current_red_envelope_activity_%d_site"
	CurrentRedEnvelopeUserListKey = "current_red_envelope_user_list_%v_site_%v_aid" // 存放能参加当前的用户

	CurrentRedEnvelopeUserIdKey = "current_red_envelope_userid_list_%d_site_%d_aid" // 存放参加当前用户ID
	RedEnvelopMessageSendKey    = "activity_red_envelope_message_aid_%d_sid_%d"     // 用于停止红包的时候 获取到红包的信息

	UserDeviceTokenCacheKey = "user_device_token_key_"
)

const (
	UNKONW_SCORE = 0
	WEB_SCORE    = 1 << iota
	H5_SCORE
	PC_SCORE
	Android_SCORE
	IOS_SCORE
	Android_SPORT_SCORE
	IOS_SPORT_SCORE
	//注意：在末尾新增类型
	END
)

type ClientType struct {
	Type  string
	Score int
}

var cliTypeM = map[string]*ClientType{}

func GetClientType(typ string) *ClientType {
	c, ok := cliTypeM[typ]
	if !ok {
		return UNKONW
	}
	return c
}

func newClientType(typ string, score int) *ClientType {
	c := &ClientType{
		Type:  typ,
		Score: score,
	}
	cliTypeM[typ] = c
	return c
}

func GetAllClientType() []*ClientType {
	list := []*ClientType{}
	for _, v := range cliTypeM {
		if v != UNKONW {
			list = append(list, v)
		}
	}

	return list
}

var (
	UNKONW        = newClientType("", UNKONW_SCORE)
	WEB           = newClientType("web", WEB_SCORE)
	H5            = newClientType("h5", H5_SCORE)
	PC            = newClientType("pc", PC_SCORE)
	Android       = newClientType("android", Android_SCORE)
	IOS           = newClientType("ios", IOS_SCORE)
	IOS_SPORT     = newClientType("ios_sport", IOS_SPORT_SCORE)
	Android_SPORT = newClientType("android_sport", Android_SPORT_SCORE)
)
