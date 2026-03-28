package base

import (
	"errors"
	"msgPushSite/lib/ws"
	"msgPushSite/mdata"
	"msgPushSite/utils"
	"strconv"
	"strings"
)

func Resp(err error, rsp *ws.Payload) {
	switch {
	case errors.Is(err, mdata.NotJoinRoomErr): // 未进入房间，参与聊天
		rsp.ResponseError(utils.ErrNotJoinRoom, err.Error())
	case errors.Is(err, mdata.UserSpeechErr): // 当前用户已被禁言
		rsp.ResponseError(utils.ErrCurrentUserMute, err.Error())
	case errors.Is(err, mdata.UserVIPLevelErr): // 未达到发言等级
		rsp.ResponseError(utils.ErrBroadcastNotAuth, err.Error())
	case errors.Is(err, mdata.RoomStatusErr): // 房间已关闭
		rsp.ResponseError(utils.ErrRoomMaintainStatus, err.Error())
	case errors.Is(err, mdata.RoomNotStartErr): // 赛事还未开始
		rsp.ResponseError(utils.ErrRoomNotStart, err.Error())
	case errors.Is(err, mdata.SerViceStatusErr): // 服务端异常
		rsp.ResponseError(utils.ErrInternal, err.Error())
	case errors.Is(err, mdata.TokenParserErr): // 登陆失败，token非法
		rsp.ResponseError(utils.ErrTokenInvalid, err.Error())
	case errors.Is(err, mdata.TokenInvalidErr): // 登陆失败，token非法
		rsp.ResponseError(utils.ErrTokenInvalid, err.Error())
	case errors.Is(err, mdata.TokenExpireErr): // token过期
		rsp.ResponseError(utils.ErrTokenExpired, err.Error())
	case errors.Is(err, mdata.ArgsParserErr): // 参数错误
		rsp.ResponseError(utils.ErrInvalidParams, err.Error())
	case errors.Is(err, mdata.UserNotLoginErr): // 用户未登录，进行发言
		rsp.ResponseError(utils.ErrAccess, err.Error())
	case errors.Is(err, mdata.RoomNotFoundErr): // 房间不存在
		rsp.ResponseError(utils.ErrNotFoundRoom, err.Error())
	case errors.Is(err, mdata.AllRoomMaintainErr): // 所有房间维护
		rsp.ResponseError(utils.ErrRoomMaintainStatus, err.Error())
	case errors.Is(err, mdata.ServiceNotInitErr): // 当前后台未进行初始化，请联系管理员
		rsp.ResponseError(utils.ErrServiceNotInit, err.Error())
	case errors.Is(err, mdata.MsgBodyEmptyErr):
		rsp.ResponseError(utils.ErrMsgBodyIsEmpty, err.Error())
	case errors.Is(err, mdata.RepeatLoginErr):
		rsp.ResponseError(utils.ErrRepeatLogin, err.Error())
	case errors.Is(err, mdata.RepeatJoinRoomErr):
		rsp.ResponseError(utils.ErrRepeatJoinRoom, err.Error())
	case errors.Is(err, mdata.GlobalRoomStatusErr):
		rsp.ResponseError(utils.ErrGlobalRoomMaintainStatus, err.Error())
	case errors.Is(err, mdata.SpeechFrequencyErr):
		rsp.ResponseError(utils.ErrSpeechFrequency, err.Error())
	case errors.Is(err, mdata.SpeechFrequencyNormalErr):
		rsp.ResponseError(utils.ErrSpeechFrequency, err.Error())
	case errors.Is(err, mdata.ShareBetAmountLimitErr):
		rsp.ResponseError(utils.ErrShareBetAmountNotEnough, err.Error())
	case errors.Is(err, mdata.ShareRecordRepeatErr):
		rsp.ResponseError(utils.ErrShareBetRecordRepeat, err.Error())
	case errors.Is(err, mdata.MsgLengthLimitExceededErr):
		rsp.ResponseError(utils.ErrMsgLimitExceeded, err.Error())
	case errors.Is(err, mdata.FrequentOperationLimitErr):
		rsp.ResponseError(utils.ErrFrequentOperationLimit, err.Error())
	default:
		if err != nil {
			errStr := err.Error()
			errList := strings.Split(errStr, "|")
			if len(errList) != 2 {
				return
			}
			code, err := strconv.Atoi(errList[0])
			if err != nil {
				return
			}
			rsp.ResponseError(code, errList[1])
		}
	}
}
