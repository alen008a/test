package ws

import (
	"bytes"
	"io"
	"msgPushSite/lib/randid"
	"msgPushSite/utils"

	"msgPushSite/mdata"
)

const (
	//default bytes buffer cap size
	defaultCapSize = 512
	//default packet pool size
	defaultPoolSize = 102400
	//default bytes buffer cap max size
	//if defaultCapSize > defaultMaxCapSize ? defaultCapSize
	defaultMaxCapSize = 4096
)

var (
	//读取标志位
	magicDelta = []byte("\r\n\t")
)

// create default pool
var pool = &packetPool{
	poolSize:   defaultPoolSize,
	capSize:    defaultCapSize,
	maxCapSize: defaultMaxCapSize,
	packets:    make(chan *Packet, defaultPoolSize),
}

type packetPool struct {
	poolSize, capSize, maxCapSize int
	packets                       chan *Packet
}

type Packet struct {
	B *bytes.Buffer
}

func newPacket() *Packet {
	return &Packet{B: bytes.NewBuffer(make([]byte, 0, pool.capSize))}
}

func Recycle(p ...*Packet) {
	for i := range p {
		p[i].B.Reset()

		if p[i].B.Cap() > pool.maxCapSize {
			p[i].B = bytes.NewBuffer(make([]byte, 0, pool.maxCapSize))
		}

		select {
		case pool.packets <- p[i]:
		default: //if pool full,throw away
		}
	}
}

func NewPacket() (p *Packet) {
	select {
	case p = <-pool.packets:
	default:
		p = newPacket()
	}
	return
}

func (p *Packet) Copy() *Packet {
	if p == nil || p.B == nil {
		return &Packet{B: bytes.NewBuffer(nil)}
	}
	data := p.B.Bytes()
	return &Packet{B: bytes.NewBuffer(append([]byte(nil), data...))}
}

func PayloadIo(body io.Reader) *Packet {
	r := NewPacket()
	r.Reset()
	_, _ = io.Copy(r.B, body)
	return r
}

func PayloadBytes(body []byte) *Packet {
	r := NewPacket()
	r.B.Reset()
	r.B.Write(body)
	return r
}

func (p *Packet) Reset() {
	p.B.Reset()
}

func (p *Packet) Release() {
	p.B.Reset()
	if p.B.Cap() > pool.maxCapSize {
		p.B = bytes.NewBuffer(make([]byte, 0, pool.maxCapSize))
	}

	select {
	case pool.packets <- p:
	default: //if pool full,throw away
	}
}

func (p *Packet) Len() int {
	return p.B.Len()
}

func (p *Packet) Write(b []byte) {
	p.B.Write(b)
}

func (p *Packet) Bytes() []byte {
	return p.B.Bytes()
}

func (p *Packet) String() string {
	return p.B.String()
}

func (p *Packet) Decode(v interface{}) error {
	return mdata.Cjson.Unmarshal(p.Bytes(), v)
}

// Packet 封包处理
func (p *Packet) Packet(msg *Msg) {
	p.Write(mdata.MustMarshal(msg))
}

// UnPacket 拆包处理
func (p *Packet) UnPacket(msg *Msg) error {
	//if !bytes.HasSuffix(p.Bytes(), magicDelta) {
	//	return errors.New("error packet")
	//}
	err := mdata.Cjson.Unmarshal(bytes.TrimSuffix(p.Bytes(), magicDelta), msg)
	if err != nil {
		return err
	}

	return nil
}

type Msg struct {
	SiteId      string      `json:"siteId"`                // 站点ID
	Seq         string      `json:"seq"`                   // 消息序
	MsgFlag     MsgFlag     `json:"msgFlag"`               // 消息标志, 用于服务端和客户端消息通信的标识,0表示只发给自己,1.针对房间号推送, 2.全局广播（配置信息）
	Key         string      `json:"name,omitempty"`        // 唯一标识, 只有单个消息推送的时候才会用到
	RoomId      string      `json:"roomId,omitempty"`      // 房间号，推送到某个具体的房间号时使用
	MsgId       uint32      `json:"msgId"`                 // 消息id，对应业务编号
	MsgData     interface{} `json:"msgData"`               // 消息体
	ClientTypes []string    `json:"clientTypes,omitempty"` // 客户端类型
	IsAgent     string      `json:"isAgent,omitempty"`     // 1: 是代理
	Trace       string      `json:"trace,omitempty"`       // 追踪号
	EsIndexName string      `json:"esIndexName,omitempty"` // 索引名称
}

type HistoryMsg struct {
	Seq         string      `json:"seq"`                   // 消息序
	MsgFlag     MsgFlag     `json:"msgFlag"`               // 消息标志, 用于服务端和客户端消息通信的标识,0表示只发给自己,1.针对房间号推送, 2.全局广播（配置信息）
	Key         string      `json:"key"`                   // 唯一标识, 只有单个消息推送的时候才会用到
	MsgId       uint32      `json:"msgId"`                 // 消息id，对应业务编号
	MsgData     interface{} `json:"msgData"`               // 消息体
	ClientTypes []string    `json:"clientTypes,omitempty"` // 客户端类型
	IsAgent     string      `json:"isAgent,omitempty"`     // 1: 是代理
	Trace       string      `json:"trace,omitempty"`       // 追踪号
}

func NewMsg(flag, id uint32) *Msg {
	return &Msg{
		Seq:     randid.GenerateId(),
		MsgFlag: flag,
		MsgId:   id,
	}
}

var (
	MsgChan chan *Packet // 消息通道
)

func SendMsgChan(msg *Packet) {
	MsgChan <- msg
}

type RspSchema struct {
	Seq       string `gorm:"column:seq" json:"seq"`             // 消息ID
	VIP       int    `gorm:"column:vip" json:"vip"`             // VIP等级
	Nickname  string `gorm:"column:name" json:"nickname"`       // nickname
	Msg       string `gorm:"column:msg" json:"msg"`             // 广播内容
	Timestamp string `gorm:"column:push_time" json:"timestamp"` // 消息到达时间
	MemberId  int    `gorm:"column:member_id" json:"memberId"`  // 用户ID
	Category  int    `gorm:"column:category" json:"category"`   // 消息类型
}

// ResEnvelopeMsgVo  红包雨对象
type ResEnvelopeMsgVo struct {
	StartTime     int64  `json:"startTime"`               //红包雨开始时间（时间戳）
	RetmainTime   int64  `json:"retmainTime"`             //红包雨持续时间（秒钟）
	RedPackId     int64  `json:"redPackId"`               //活动ID
	Session       int64  `json:"session,omitempty"`       //活动场次
	CountDownTime int64  `json:"countDownTime,omitempty"` //倒计时时间（分钟）
	Style         string `json:"style,omitempty"`         //红包样式CODE
	Status        int64  `json:"status,omitempty"`        //红包雨状态（1-启动，4-停止）
	CurrentTime   int64  `json:"currentTime,omitempty"`   //服务器当前时间（时间戳）
	GamePlatform  string `json:"gamePlatform,omitempty"`  //游戏设备(1-全站APP，2-体育APP)
	Type          int    `json:"type,omitempty"`          // 红包雨对象（1-全站，0-个人）
	SiteId        int64  `json:"siteId,omitempty"`
}

type Payload struct {
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
}

func (p *Payload) ResponseSetData(data interface{}) {
	p.Data = data
}

func (p *Payload) ResponseError(code int, msg string) {
	p.StatusCode = code
	p.Message = msg
}

func (p *Payload) ResponseOK(data interface{}) {
	p.StatusCode = utils.StatusOK
	p.Message = utils.MsgSuccess
	p.Data = data
}

type BroadcastSchema struct {
	Msg  *Msg
	Self []string
	buff []byte
}
