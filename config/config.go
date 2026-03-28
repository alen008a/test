package config

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"gopkg.in/ini.v1"
	"msgPushSite/mdata"
	"msgPushSite/mdata/namespace"
	"strings"
)

var g *Configurations

const (
	GroupGlobal = "Global"
)

// Configurations 存放当前的配置
type Configurations struct {
	// 日志配置
	Logger Logger
	// 应用配置
	Application
	// Redis读写库
	RedisCore Redis
	Live      Mysql
	LiveSlave Mysql

	Kafka Kafka

	KafkaTopic KafkaTopic

	// 服务地址配置
	ServiceAddr

	ChatMessageLimit

	EsSite ChatES

	ChatMsgReportWarn

	ago    config_client.IConfigClient
	global config_client.IConfigClient
	f      map[namespace.NacosNamespace]interface{}
	app    NacosConfig
}

// 获取当前环境
func GetEnv() string {
	return strings.ToLower(g.app.Env)
}

func GetConfig() *Configurations {
	return g
}

func GetKafkaConfig() *Kafka {
	return &g.Kafka
}

func GetKafkaTopic() *KafkaTopic {
	return &g.KafkaTopic
}

func GetServiceAddr() ServiceAddr {
	return g.ServiceAddr
}

func GetLive() *Mysql {
	return &g.Live
}
func GetLiveSlave() *Mysql {
	return &g.LiveSlave
}

func GetRWRedisConfig() *Redis {
	return &g.RedisCore
}

func GetApplication() *Application {
	return &g.Application
}

func GetChatMessageLimitConfig() *ChatMessageLimit {
	return &g.ChatMessageLimit
}

func GetChatESConfig() *ChatES {
	return &g.EsSite
}

func GetChatMsgReportWarnConfig() *ChatMsgReportWarn {
	return &g.ChatMsgReportWarn
}

type Logger struct {
	LogPath  string
	LogLevel string
	LogType  string
}

type NacosConfig struct {
	Nacos `ini:"Nacos"`
}

// Application 服务基本信息配置
type Application struct {
	Env                      string
	Address                  string
	Port                     string
	Cluster                  string
	DBSecretKey              string
	WarningCode              string
	AppID                    string
	VerifyCodeDomain         string
	ProxyURL                 string
	VerifyProxyUrl           string
	RecordRunLogTimeOpen     bool // 记录运行日志开关
	SiteAliIpUrlOpen         bool // 是否需要切换获取IP源
	OpenMiddlewareTraceDebug bool
	AutoloadConfig           bool // 项目配置热加载开关，true为热加载。
}

type ServiceAddr struct {
	IPConfAddr     string
	IP2RegionAddr  string // 新增ip库地址配置
	SiteIpv6Url    string // ipv6请求API
	SiteIpv6Key    string // ipv6ApiKey
	SiteAliIpUrlV2 string // 切换新ipv4/ipv6地址源
	SiteAliIPKeyV2 string // 切换新ipv4/ipv6地址源 key
	SiteAliIpv4Url string // aliyun ipv4解析
	SiteAliIpv4Key string // aliyun ipv4解析
}

type Mysql struct {
	Address     string
	LogEnable   bool
	IdleConnect int
	MaxConnect  int
	MaxLifeTime int
}

type Redis struct {
	Host     string //如果是集群模式，配置的是节点的host:port,如果是哨兵模式，配置的是哨兵节点的host:port
	Auth     string
	Master   string //哨兵模式必配，其他模式不用配
	PoolSize int
}

type Kafka struct {
	KafkaAddr string
}

type KafkaTopic struct {
	ChatMsgWriteTopic string
	ChatMsgBackTopic  string
}

type ChatMessageLimit struct {
	SendTimeLimit int64
	Switch        bool
}

type ChatES struct {
	Address    string
	AuthUser   string
	AuthPass   string
	TidbSwitch bool
	ESSwitch   bool
}

type ChatMsgReportWarn struct {
	Platform    string
	Env         string
	Switch      bool
	ServiceCode string
}

// Nacos 从本地ini文件读取Nacos配置
type Nacos struct {
	Env         string `ini:"Env"`         //环境
	NamespaceId string `ini:"NamespaceId"` //命名空间Id
	AppID       string `ini:"AppID"`       //应用id
	Address     string `ini:"Address"`     //Nacos地址
	Port        uint64 `ini:"Port"`        //端口号
	Scheme      string `ini:"Scheme"`      //协议
	Username    string `ini:"Username"`    //服务端的API鉴权Username
	Password    string `ini:"Password"`    //服务端的API鉴权Password
	CacheDir    string `ini:"CacheDir"`    //缓存service信息的目录，默认是当前运行目录
	LogDir      string `ini:"LogDir"`      //日志存储路径
}

func GetApp() NacosConfig {
	return g.app
}

// InitConfig 初始化配置
func InitConfig(confPath string) error {
	apo := new(NacosConfig)
	if err := ini.MapTo(apo, confPath); err != nil {
		return err
	}

	g = new(Configurations)
	g.app = *apo
	g.f = make(map[namespace.NacosNamespace]interface{})

	err := InitNacos()
	if err != nil {
		return err
	}
	err = g.loadConfig()
	if err != nil {
		return err
	}
	// 读apollo配置
	if GetConfig().AutoloadConfig {
		fmt.Println("开启动态加载")
		go g.poller()
	} else {
		fmt.Println("未开启动态加载")
	}
	return nil
}

func (c *Configurations) loadConfig() error {
	err := g.decode(namespace.Logger, &g.Logger)
	if err != nil {
		return err
	}
	err = g.decode(namespace.Application, &g.Application)
	if err != nil {
		return err
	}
	err = g.decode(namespace.Kafka, &g.Kafka)
	if err != nil {
		return err
	}
	err = g.decode(namespace.KafkaTopic, &g.KafkaTopic)
	if err != nil {
		return err
	}
	err = g.decode(namespace.EsSite, &g.EsSite)
	if err != nil {
		return err
	}
	err = g.decode(namespace.Live, &g.Live)
	if err != nil {
		return err
	}
	err = g.decode(namespace.LiveSlave, &g.LiveSlave)
	if err != nil {
		return err
	}
	err = g.decode(namespace.Redis, &g.RedisCore)
	if err != nil {
		return err
	}
	err = g.decode(namespace.ServiceAddr, &g.ServiceAddr)
	if err != nil {
		return err
	}
	err = g.decode(namespace.ChatMessageLimit, &g.ChatMessageLimit)
	if err != nil {
		return err
	}
	err = g.decode(namespace.ChatMsgReportWarn, &g.ChatMsgReportWarn)
	if err != nil {
		return err
	}
	return nil
}

func InitNacos() error {
	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(g.app.NamespaceId), //当namespace是public时，此处填空字符串。
		constant.WithTimeoutMs(90000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir(g.app.LogDir),
		constant.WithUsername(g.app.Username),
		constant.WithPassword(g.app.Password),
		constant.WithCacheDir(g.app.CacheDir),
	)
	serverConfigs := []constant.ServerConfig{
		*constant.NewServerConfig(
			g.app.Address,
			g.app.Port,
			constant.WithScheme(g.app.Scheme)),
	}
	var err error
	g.ago, err = clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return err
	}

	clientGlobalConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(GroupGlobal),
		constant.WithTimeoutMs(90000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir(g.app.LogDir),
		constant.WithUsername(g.app.Username),
		constant.WithPassword(g.app.Password),
		constant.WithCacheDir(g.app.CacheDir),
	)
	g.global, err = clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientGlobalConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *Configurations) remoteCfg(ns namespace.NacosNamespace) (string, error) {
	if strings.HasPrefix(ns, GroupGlobal) { // 公共配置 Global.Database.xxx
		split := strings.Split(ns, ".")
		return c.global.GetConfig(vo.ConfigParam{DataId: ns, Group: split[1]})
	}

	// 服务配置 命名空间名称.服务名称.配置(Api.gameSite.Application)
	return c.ago.GetConfig(vo.ConfigParam{DataId: ns, Group: c.app.AppID})
}

func (c *Configurations) decode(ns namespace.NacosNamespace, v interface{}) error {
	// 注册到函数里
	if !strings.HasPrefix(ns, GroupGlobal) && !strings.ContainsAny(ns, ".") {
		ns = c.app.AppID + "." + ns
	}

	content, err := c.remoteCfg(ns)
	if err != nil {
		fmt.Printf("加载配置失败！配置名称：%v \n", ns)
		return err
	}

	fmt.Println("------------------------配置读取成功------------------", ns, content)
	c.f[ns] = v
	err = mdata.Cjson.UnmarshalFromString(content, v)
	if err != nil {
		return err
	}
	c.f[ns] = v
	return err
}

func (c *Configurations) onChange(ns namespace.NacosNamespace, v interface{}) error {
	err := c.ago.ListenConfig(vo.ConfigParam{DataId: ns, Group: "DEFAULT_GROUP", OnChange: func(namespace, group, dataId, data string) {
		oldValue, _ := mdata.Cjson.MarshalToString(v)
		fmt.Println("-----------------监听配置变动--原有配置------------ group:" + group + ", dataId:" + dataId + ", content:" + oldValue)
		err := mdata.Cjson.UnmarshalFromString(data, v)
		if err != nil {
			return
		}
		fmt.Println("-----------------监听配置变动--最新配置------------ group:" + group + ", dataId:" + dataId + ", content:" + data)
	}})
	if err != nil {
		return err
	}
	return nil
}

func (c *Configurations) poller() {
	go func() {
		for ns, v := range c.f {
			oldValue := v
			err := c.onChange(ns, v)
			if err != nil {
				fmt.Printf("Dynamic update |namespace=%s\n |old=%+v\n |new=%+v\n |err=%v\n", ns, oldValue, v, err)
			}
		}
	}()
}
