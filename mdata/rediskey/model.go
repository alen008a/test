package rediskey

// 红包雨活动红包等级
type ActivityEnvelopeValue struct {
	Id            int64   `gorm:"column:id" json:"id"`                                      // 主键
	ActivityId    int     `gorm:"column:activity_id;default:0" json:"activityId"`           // 活动ID
	IsDelete      int     `gorm:"column:is_delete;default:0" json:"isDelete"`               // 是否删除(1-删除,0-不删除)
	CreatedAt     string  `gorm:"column:created_at;default:null" json:"createdAt"`          // 创建时间
	CreatedBy     string  `gorm:"column:created_by" json:"-"`                               // 创建人
	UpdatedAt     string  `gorm:"column:updated_at;default:null" json:"updatedAt"`          // 最后一次更新时间
	UpdatedBy     string  `gorm:"column:updated_by" json:"-"`                               // 最后一次操作人
	Name          string  `gorm:"column:name" json:"name"`                                  // 红包名称
	MinValue      float64 `gorm:"column:min_value;default:0.00" json:"minValue"`            // 红包最小金额
	MaxValue      float64 `gorm:"column:max_value;default:0.00" json:"maxValue"`            // 红包最大金额
	MaxQuantity   int     `gorm:"column:max_quantity;default:0" json:"maxQuantity"`         // 红包最大数量
	VipLevel      string  `gorm:"column:vip_level" json:"vipLevel"`                         // 会员VIP等级限定
	Key           string  `gorm:"column:key" json:"key"`                                    // 前端使用
	OddsOfWinning float64 `gorm:"column:odds_of_winning;default:0.00" json:"oddsOfWinning"` // 中奖概率
	IsVip         int     `gorm:"column:is_vip;default:0" json:"isVip"`                     // 是否是VIP群组：0-不是，1-是
	GroupName     string  `gorm:"column:group_name" json:"groupName"`                       // 群组名称
	UserList      string  `gorm:"column:user_list" json:"userList"`                         // 自定义群组用户列表
}

func (*ActivityEnvelopeValue) TableName() string {
	return "activity_envelope_value"
}

// ActivityEnvelopeVo  当前可以使用的红包雨信息
type ActivityEnvelopeVo struct {
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

// 红包雨活动
type ActivityEnvelope struct {
	Id                  int64   `gorm:"column:id" json:"id"`                                                  // 主键
	SiteId              int     `gorm:"column:site_id" json:"siteId"`                                         // 站点id
	ActivityId          int64   `gorm:"column:activity_id;default:0" json:"activityId"`                       // 活动ID
	MemberFilterId      int     `gorm:"column:member_filter_id;default:0" json:"memberFilterId"`              //会员筛选id type ==0 时候需要传
	Type                int     `gorm:"column:type;default:0" json:"type"`                                    // 红包雨对象（1-全站，0-个人）
	StyleType           int     `gorm:"column:style_type;default:1" json:"styleType"`                         // 红把手样式代码 1为默认类型 2 为自定义
	Style               string  `gorm:"column:style" json:"style"`                                            // 红包样式编号（PTHB，YDHB...）
	AccountClaim        int     `gorm:"column:account_claim" json:"accountClaim"`                             // 账号要求（0-无要求，1、所有要求，2-绑定手机号，3-绑定银行卡） 指针类型 方便更新空字符串
	VipLevel            string  `gorm:"column:vip_level" json:"vipLevel"`                                     // 抢红包等级
	Countdown           int     `gorm:"column:countdown;default:0" json:"countdown"`                          // 每次红包雨倒计时（1-10整数）
	EnvelopeLimit       int     `gorm:"column:envelope_limit;default:0" json:"envelopeLimit"`                 // 每次抢红包个数上限
	EnvelopeDuration    int64   `gorm:"column:envelope_duration;default:0" json:"envelopeDuration"`           // 每次红包时长
	PrizePoolLimit      float64 `gorm:"column:prize_pool_limit;default:0.00" json:"prizePoolLimit"`           // 奖池上限
	CumulativeValidBets float64 `gorm:"column:cumulative_valid_bets;default:0.00" json:"cumulativeValidBets"` // 累计有效投注
	FlowRequirements    int     `gorm:"column:flow_requirements;default:0" json:"flowRequirements"`           // 流水倍数要求（1~10000整数）
	FloatLayerWeb       string  `gorm:"column:float_layer_web" json:"floatLayerWeb"`                          // WEB红包浮层
	FloatLayerH5        string  `gorm:"column:float_layer_h5" json:"floatLayerH5"`                            // H5红包浮层
	GamePlatform        string  `gorm:"column:game_platform" json:"gamePlatform"`                             // 游戏设备(1-全站APP，2-体育APP)
	JgType              *int    `gorm:"column:jg_type;default:0" json:"jgType"`                               // 极光类型（0-默认内容，1-自订） 指针类型 方便更新0值
	JgTitle             string  `gorm:"column:jg_title" json:"jgTitle"`                                       // 极光推送标题
	JgContent           string  `gorm:"column:jg_content" json:"jgContent"`                                   // 极光推送内容
	ConfigType          int     `gorm:"column:config_type;default:0" json:"configType"`                       // 红包金额配置类型：0-奖品导向模板，1-用户导向模板   指针类型 方便更新0值
	IsDelete            int     `gorm:"column:is_delete;default:0" json:"isDelete"`                           // 是否删除(1-删除,0-不删除)
	CreatedAt           string  `gorm:"column:created_at" json:"createdAt"`                                   // 创建时间
	CreatedBy           string  `gorm:"column:created_by" json:"createdBy"`                                   // 创建人
	UpdatedAt           string  `gorm:"column:updated_at" json:"updatedAt"`                                   // 最后一次更新时间
	UpdatedBy           string  `gorm:"column:updated_by" json:"updatedBy"`                                   // 最后一次操作人
	WhitelistMember     string  `gorm:"column:whitelist_member" json:"whitelistMember"`                       // 白名单会员列表(多个用逗号隔开)
	WalletType          int     `gorm:"column:wallet_type;default:0" json:"walletType"`                       // 钱包类型：1-中心钱包，2-场馆钱包
	ActivityVenues      string  `gorm:"column:activity_venues" json:"activityVenues"`                         // 活动场馆  默认人民币;AGZR-AG真人;EBETZR-EBET真人;IMQP-IM棋牌;SGCP-双赢彩票;TCGCP-TCG彩票;IMDJ-IM电竞;IMTY-IM体育;AGBY-AG捕鱼;PPDZ-PP电子;
}
