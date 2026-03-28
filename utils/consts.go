package utils

const (
	HeaerToken = "X-API-TOKEN"
	ClientType = "Client-Type"

	// 客户端类型
	WebClientType          = "web"
	H5ClientType           = "h5"
	AndroidClientType      = "android"
	AndroidSportClientType = "android_sport"
	IOSClientType          = "ios"
	IOSSportClientType     = "ios_sport"
	AndroidChessClientType = "android_chess"
	IOSChessClientType     = "ios_chess"
	WebManagerType         = "web_manager"

	// 消息类型
	AllMsgType            = "1"
	WebMsgType            = "3"
	H5MsgType             = "4"
	AndroidMsgType        = "5"
	AndroidSportMsgType   = "6"
	IOSMsgType            = "7"
	IOSSportMsgType       = "8"
	IOSChessMsgType       = "9"
	AndroidChessMsgType   = "10"
	ForbiddenChessMsgType = "11" //除开棋牌推送的其他所有

	// 客户端类型
)

var (
	// ClientTypeMap 客户端类型
	ClientTypeMap = map[string]string{
		WebClientType:          "3",
		H5ClientType:           "4",
		AndroidClientType:      "5",
		AndroidSportClientType: "6",
		IOSClientType:          "7",
		IOSSportClientType:     "8",
		IOSChessClientType:     "9",
		AndroidChessClientType: "10",
		WebManagerType:         "11",
	}

	// IpServerClientTypeMap 客户端类型
	IpServerClientTypeMap = map[string]string{
		AndroidClientType:      "5",
		AndroidSportClientType: "6",
		IOSClientType:          "7",
		IOSSportClientType:     "8",
	}
	// MsgTypeMap 消息类型
	MsgTypeMap = map[string]string{
		AllMsgType:            "1",
		WebMsgType:            "3",
		H5MsgType:             "4",
		AndroidMsgType:        "5",
		AndroidSportMsgType:   "6",
		IOSMsgType:            "7",
		IOSSportMsgType:       "8",
		IOSChessMsgType:       "9",
		AndroidChessMsgType:   "10",
		ForbiddenChessMsgType: "11",
	}
)
