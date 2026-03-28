package rediskey

const (
	MemberSpeechStatus         = "member_speech_status_%s"          // set 会员是否存在黑名单
	MemberSpeechBannedDuration = "member_speech_banned_duration_%s" //禁言时长 开始时间
	ChatVIPConf                = "chat_vip_conf_%s"                 // 获取VIP相关配置
	LiveMatch                  = "live_match_%s_%s"                 // 获取房间信息
	LiveMatchMessage           = "live_match_message_%s_%s"         // 历史聊天记录
	AllMatchBan                = "all_match_ban_%s"                 // 所有房间维护
	DoubleLiveMatchMsg         = "double_live_match_msg_%s_%s_%s"   // 获取房间分享注单单双重缓存
	LiveTotal                  = "live_total_%s_%s"                 // 聊天室消息总数缓存
	LiveRoomActiveTotal        = "live_room_active_total_%s"        // 聊天室在线用户
	LiveRoomEnterTotal         = "live_room_enter_total_%s"         // 聊天室进线用户

	MemberNameRoomBind = "member_name_bind_%s_%s_%s" // set 记录当前账号在当前房间所有登陆的连接ID
	MemberGlobalCID    = "member_global_cid_%s_%s"   // set 记录当前账号所有加入的房间

	ShareBetsAmountLimit       = "share_bets_amount_limit_%s"
	SettleShareBetsAmountLimit = "settle_share_bets_amount_limit_%s"
	ShareBetsBigAmountLimit    = "share_bets_big_amount_limit_%s"
	ShareBetRecordRepeatLimit  = "share_bet_record_repeat_%s_%s" // 重复晒单

	ChangeRoomLimit          = "change_room_limit_%s"             // 切换房间限制
	GetHistoryRecordLimit    = "get_history_record_limit_%s_%s"   // 获取历史聊天记录限制
	BulletButtonSetting      = "bullet_button_setting_%s"         // 是否关闭弹幕按钮 显示
	BulletShowSetting        = "bullet_show_setting_%s"           // 是否默认开启弹幕 显示
	BetsStrategiesOpen       = "bets_strategies_open_%s"          // 推单开关
	AdvStrategiesOpen        = "adv_strategies_open_%s"           //返回广告banner 开关状态
	LiveGiftStatusOpen       = "live_gift_status_open_%s"         // 礼物打赏开关
	ChatIsCanCopyContentOpen = "chat_is_can_copy_content_open_%s" //是否可否之内容开关
	ChatMatchScoreStatusOpen = "chat_match_score_status_open_%s"  //比分预测显示开关

	SiteActiveMemberInfo = "active_member_info_%s"
	ClientMemberName     = "client_member_name_%s"

	YSeriesJumpUrl = "y_series_jump_url_%s"

	ShieldNumberSet  = "shield_number_set" //消息位数屏蔽设置
	SensitiveTable   = "sensitive_word"
	SensitivePublish = "sensitive_publish"

	// HasRedEnvelopeSessionKey 红包雨相关的操作
	//HasRedEnvelopeSessionKey      = "has_red_envelope_session_%d_site" // 查看是否有全站的红包雨
	//CurrentRedEnvelopeActivityKey = "current_red_envelope_activity_%d_site"
	//CurrentRedEnvelopeUserListKey = "current_red_envelope_user_list_%v_site_%d_aid"   // 存放能参加当前的用户
	//CurrentRedEnvelopeUserIdKey   = "current_red_envelope_userid_list_%d_site_%d_aid" // 存放参加当前用户ID
)
