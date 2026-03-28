package utils

type StatusCode = int

const (
	StatusOK                    = 6000 // 正常返回
	ErrAccess                   = 6001 // 登录状态失效 类似401, 会跳到登录窗口
	ErrAccount                  = 6002 // 账号或密码错误
	ErrRefuse                   = 6003 // 访问被拒绝类似403
	ErrNotFound                 = 6004 // 找不到接口类似404
	ErrInternal                 = 6005 // 接口服务器产生错误 类似500
	ErrInvalidGateway           = 6006 // 无效网关 类似502
	ErrAPIFailed                = 6007 // 接口请求失败, 无法接受参数，会跳到登录窗口
	ErrInvalidParams            = 6008 // 违法的参数
	ErrDataExistFailed          = 6009 // 数据已经存在
	ErrFileUploadFailed         = 6010 // 文件上传失败
	ErrLockedFailed             = 6011 // 频繁提交拒绝
	ErrWarning                  = 6012 // 警告
	ErrMaintenance              = 6013 // 维护
	ErrCallError                = 6014 // 调用错误
	ErrNotBindPhone             = 6015 // 未绑定手机号的错误码
	ErrLoginVerify              = 6016 // 用户登录需要手机验证
	ErrLoginVerify2             = 6017 // 用户登录需要手机验证
	ErrWalletAccess             = 6099 // 钱包token失效
	ErrGraphicVerification      = 6021 // 图形验证码
	ErrGeetestVerification      = 6022 // 极速验证
	ErrIPNotAllowLogin          = 6025 // IP不通过
	ErrDeviceNotAllowLogin      = 6026 // 设备不通过
	ErrNotAllowLogin            = 6027 // 设备IP 均不通过校验
	ErrLongTimeNotLogin         = 6028 // 长时间未登陆
	ErrRiskLoginNotAllow        = 6029 // 登陆存在风险 不允许直接登陆 需要手机号验证
	ErrContactCustomerService   = 6030 // 联系客服
	ErrTokenExpired             = 6031 // token过期
	ErrTokenInvalid             = 6032 // token校验失败
	ErrBroadcastNotAuth         = 6033 // 当前没有聊天权限
	ErrRoomMaintainStatus       = 6034 // 房间状态停用
	ErrRoomNotStart             = 6035 // 赛事还未开启
	ErrCurrentUserMute          = 6036 // 当前用户已被禁言
	ErrNotJoinRoom              = 6037 // 未加入房间，进行聊天
	ErrNotFoundRoom             = 6038 // 房间不存在
	ErrServiceNotInit           = 6039 // 配置信息未进行初始化
	ErrMsgBodyIsEmpty           = 6040 // 消息体不能为空
	ErrRepeatLogin              = 6041 // 重复登陆
	ErrRepeatJoinRoom           = 6042 // 重复加入房间
	ErrGlobalRoomMaintainStatus = 6043 // 所有房间都在维护
	ErrSpeechFrequency          = 6044 // 当前VIP等级发言过于频繁
	ErrShareBetRecordNotEnough  = 6045 // 晒单金额不足
	ErrShareBetAmountNotEnough  = 6046 // 未结算注单分享限制
	ErrShareBetRecordRepeat     = 6047 // 晒单多次重复
	ErrMsgLimitExceeded         = 6048 // 发送消息长度超出150个字符限制
	ErrFrequentOperationLimit   = 6049 // 操作过于频繁，请稍后重试
)

//只能写通用的msg，非通用的，直接在c.WebRsp里面返回

type Message = string

const (
	MsgSuccess              Message = "成功"
	MsgInternalError                = "服务器错误"
	MsgAccessError                  = "请重新登录"
	MsgFileUploadError              = "文件上传失败"
	MsgInvalidParamsError           = "请求参数错误"
	MsgTokenInvalidErr              = "登陆未成功"
	MsgRefuseError                  = "请求拒绝"
	MsgLockedError                  = "请勿频繁提交"
	MsgMaintenanceError             = "维护中"
	MsgNotFoundError                = "未找到数据"
	MsgIllegalRequestError          = "非法请求" //请求头未传加密串X-API-TIMESTAMP的时候使用
	MsgLoginForbidAreaLimit         = "地区ip限制,不允许登陆"
	MsgBroadcastNotAuth             = "当前没有参与聊天权限"
	MsgTimeOut                      = "请求超时"
	MsgIllegalSiteError             = "未知站点"
)
