package namespace

type NacosNamespace = string

const (
	Application       NacosNamespace = "Application"
	Logger                           = "Logger"
	ServiceAddr                      = "ServiceAddr"
	ChatMessageLimit                 = "ChatMessageLimit"
	ChatMsgReportWarn                = "ChatMsgReportWarn"
	Kafka                            = "Global.MQ.Kafka"
	EsSite                           = "Global.ES.EsSite"
	KafkaTopic                       = "Global.Config.KafkaTopic"
	Redis                            = "Global.Redis.RedisCore"
	Live                             = "Global.Database.LiveDb"
	LiveSlave                        = "Global.Database.LiveSlaveDb"
)
