package ws

import (
	ctx "context"
	"encoding/json"
	"fmt"
	"msgPushSite/config"
	"msgPushSite/internal/glog"
	"msgPushSite/lib/es"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"msgPushSite/lib/randid"
	"msgPushSite/mdata"
	"msgPushSite/utils"

	"github.com/gorilla/websocket"
)

// Client is a middleman between the ws connection and the hub.
type Client struct {
	Id             string            // 唯一标识
	siteId         string            // 站点ID
	hub            *Hub              // 归属在哪个Hub
	conn           *websocket.Conn   // ws连接
	send           chan []byte       // ws writer消息缓冲
	roomID         string            // 房间号
	room           *Room             // 和哪个房间关联
	key            string            // 客户端唯一标志 登录后为用户名
	clientType     string            // 客户端类型
	clientIp       string            // 客户端ip
	state          uint32            // 连接状态
	member         *mdata.MemberInfo // 用户信息
	server         string            // 暂时用于记录用户所在服务器内网ip
	userAgent      string            // userAgent
	mux            sync.RWMutex      // 锁
	isClosed       int32             // 连接是否关闭
	ctx            ctx.Context       // 上线文管理
	cancel         ctx.CancelFunc    // 关闭调用
	isLogin        bool              // 是否登陆
	creatAt        int64             // 连接创建时间 扫描要用
	ip             string            // IP地址
	RoomPushMethod PushToKafka
	property       map[string]interface{} // 属性
}

func (c *Client) SetProperty(key string, value interface{}) {
	c.mux.Lock()
	c.property[key] = value
	c.mux.Unlock()
}

func (c *Client) GetProperty(key string) (res interface{}, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	res, ok = c.property[key]
	return
}

func (c *Client) UserAgent() string {
	return c.userAgent
}

func (c *Client) GetSiteId() string {
	return c.siteId
}

func (c *Client) IsAgent() string {
	if c.member == nil {
		return ""
	}
	return c.member.IsAgent
}

func (c *Client) SetLogin() {
	c.isLogin = true
}

func (c *Client) GetLogin() bool {
	return c.isLogin
}

func (c *Client) Ip() string {
	return c.ip
}

func (c *Client) GetUsername() string {
	return c.key
}

// Member 函数里面必须调用判断用户是否登录
func (c *Client) Member() (*mdata.MemberInfo, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if c.member == nil {
		return nil, mdata.UserNotLoginErr
	}

	return c.member, nil
}

func (c *Client) ClientType() string {
	return c.clientType
}

func (c *Client) GetRoom() *Room {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.room
}

func (c *Client) SetRoom(room *Room) {
	c.mux.Lock()
	c.room = room
	if room != nil {
		c.roomID = room.id
	}
	c.mux.Unlock()
}

func (c *Client) setState(s state) {
	atomic.SwapUint32(&c.state, s)
}

func (c *Client) getState() state {
	return atomic.LoadUint32(&c.state)
}

func (c *Client) Close() {

	glog.Infof("[关闭连接] close client  siteId:%v hub:%v platform:%v", c.siteId, c.hub.id, c.clientType)
	if c.member != nil {
		glog.Infof("[关闭连接] close client  siteId:%v hub:%v platform:%v name:%v", c.siteId, c.hub.id, c.clientType, c.member.Name)
	}

	if atomic.CompareAndSwapInt32(&c.isClosed, 0, 1) == false {
		return
	}
	app.onConnectionStop(c)
	c.cancel()
	c.hub.unregister <- c
	_ = c.conn.Close()
}

func (c *Client) IsClose() bool {
	if atomic.LoadInt32(&c.isClosed) == 1 {
		return true
	}
	return false
}

func (c *Client) release() {

	glog.Infof("[释放连接] release client  siteId:%v hub:%v platform:%v", c.siteId, c.hub.id, c.clientType)
	if c.member != nil {
		glog.Infof("[释放连接] release client  siteId:%v hub:%v platform:%v name:%v", c.siteId, c.hub.id, c.clientType, c.member.Name)
	}

	if c.room != nil {
		c.room.Remove(c.Id)
	}
	if c.hub != nil {
		c.hub.Remove(c.Id)
	}
	c.room = nil
	c.hub = nil
	c.isLogin = false
	c.member = nil
}

func (c *Client) readPumpWss() {
	defer func() {
		c.Close()
		if err := recover(); err != nil {
			fmt.Println(err)

			glog.Infof("[readPumpWss 出错] readPumpWss client  siteId:%v hub:%v platform:%v", c.siteId, c.hub.id, c.clientType)
			if c.member != nil {
				glog.Infof("[readPumpWss 出错] readPumpWss client  siteId:%v hub:%v platform:%v name:%v", c.siteId, c.hub.id, c.clientType, c.member.Name)
			}

			glog.Emergency("ws client readPumpWss panic recover error|err=>%v", err)
		}
	}()

	c.conn.SetReadLimit(MaxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	wsContext := NewWsContext()
	wsContext.Client = c
	wsContext.SiteId = c.GetSiteId()

	//同一个客户端的消息不存在并发，串行执行即可
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if wsContext.IsClose() {
				return
			}
			messageType, body, err := c.conn.NextReader()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					//todo 调试暂时打开日志
					wsContext.Errorf("readPumpWss |err=%v", err)
				}
				return
			}

			//底层会默认注册心跳包,心跳不做处理
			switch messageType {
			case websocket.PingMessage:
				continue
			case websocket.CloseMessage:
				return
			}

			var packet, msg = PayloadIo(body), Msg{}

			err = packet.UnPacket(&msg)
			if err != nil {
				wsContext.Errorf("readPumpWss |err=%v |msg=%v |key=%s", err, string(mdata.MustMarshal(&msg)), c.key)
				packet.Release()
				continue
			}

			endpoint, err := getEndpoint(msg.MsgId)

			if err != nil || endpoint == nil {
				wsContext.Errorf("unknown MsgId %d |err=%v |key=%s", msg.MsgId, err, c.key)
				packet.Release()
				continue
			}

			var payload = Payload{
				StatusCode: utils.StatusOK,
				Message:    utils.MsgSuccess,
			}

			if msg.MsgId == mdata.MsgIdRedPackageReceive {

			}

			// 考虑到广播比较耗时 需要其他的路由处理 里面使用go func 说需要对包进行copy
			if msg.MsgId == mdata.MsgIdRedPackageRain {
				newPackage := packet.Copy()
				msg.MsgFlag = MiddleApply(endpoint)(wsContext, newPackage, &payload, &msg)
				packet.Release()
				continue
			}

			if msg.MsgId == mdata.MsgIdRedPackageReceive {
				newPackage := packet.Copy()
				msg.MsgFlag = MiddleApply(endpoint)(wsContext, newPackage, &payload, &msg)
				packet.Release()
				continue
			}

			packet.Reset()
			packet.Write(mdata.MustMarshal(msg.MsgData))

			// 当未登录时，name为空
			// 当用户登录后，会自动在后面的报文中赋值
			msg.Key = c.key
			msg.SiteId = c.GetSiteId()
			if room := c.GetRoom(); room != nil {
				msg.RoomId = c.GetRoom().GetID()
			}
			msg.EsIndexName = fmt.Sprintf(es.ESIndexPrefix, c.GetSiteId()) + time.Now().Format("2006_01")
			if err != nil {
				wsContext.Errorf("readPumpWss |err=%v |msg=%v |key=%s", err, mdata.MustMarshal(&msg), c.key)
				//传送了异常的协议id
				packet.Release()
				continue
			}

			if len(msg.Seq) == 0 {
				msg.Seq = randid.GenerateId()
			}

			inPkt := packet.Copy()
			msg.MsgFlag = MiddleApply(endpoint)(wsContext, inPkt, &payload, &msg)
			inPkt.Release()

			//对象重用
			packet.Reset()
			msg.MsgData = payload
			switch msg.MsgFlag {
			case MsgFlagSelf: // 推送给自己

				if strings.Count(msg.RoomId, "_") >= 2 {
					msg.RoomId = msg.RoomId[strings.Index(msg.RoomId, "_")+1:]
				}
				packet.Packet(&msg)

				_ = c.trySend(packet.Bytes(), 50*time.Millisecond) // ⭐拷贝+短超时

			case MsgFlagRoom: // 在房间内广播，推送到kafka
				preHandleMsg(&msg)
				packet.Packet(&msg)

				if msg.MsgFlag == MsgFlagSelf {
					if strings.Count(msg.RoomId, "_") >= 2 {
						msg.RoomId = msg.RoomId[strings.Index(msg.RoomId, "_")+1:]
					}

					// 直接推给客户端
					_ = c.trySend(packet.Bytes(), 50*time.Millisecond) // ⭐拷贝+短超时

					// Kafka 仅用于入库：也要拷贝
					app.RoomPushMethod(utils.Clone(packet.Bytes()), config.GetKafkaTopic().ChatMsgWriteTopic) // ⭐
				} else {

					app.RoomPushMethod(utils.Clone(packet.Bytes()), config.GetKafkaTopic().ChatMsgWriteTopic) // ⭐
				}
			}
			packet.Release()
		}
	}
}

// 统一安全发送（短超时+踢慢端 或丢弃）
func (c *Client) trySend(b []byte, wait time.Duration) bool {

	data := utils.Clone(b) // **关键：深拷贝**
	select {
	case c.send <- data:
		return true
	case <-time.After(wait):
		// 选择其一：踢慢端 或 仅记录丢弃
		// c.Close()
		glog.Warnf("send timeout, drop |key=%s", c.key)
		return false
	}
}

func (c *Client) trySendNoClone(b []byte, wait time.Duration) bool {

	select {
	case c.send <- b:
		return true
	case <-time.After(wait):
		glog.Warnf("send timeout, drop |key=%s", c.key)
		return false
	}
}

func (c *Client) writePumpWss() {

	var timeoutChan = make(chan struct{})

	var tw = mdata.TimingWheel.ScheduleFunc(
		&mdata.RotateScheduler{Interval: pingPeriod}, func() {
			if err := c.sendMsg(nil, websocket.PingMessage); err != nil {
				glog.Infof("send ping message error|err=>%v", err)
				timeoutChan <- struct{}{}
			}
		},
	)

	defer func() {
		tw.Stop()
		c.Close()
	}()

	for {
		select {
		case data, ok := <-c.send:
			if !ok {
				break
			}
			if !json.Valid(data) {

				glog.Errorf("最终发送数据格式 非json格式 原始数据:%s", string(data))
			}
			err := c.sendMsg(data, websocket.TextMessage)
			if err != nil {
				return
			}
			go c.backMsgToKafka(data)
		case <-timeoutChan:
			return
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) sendMsg(data []byte, wsType int) error {
	c.mux.Lock()
	defer c.mux.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	w, err := c.conn.NextWriter(wsType)
	if err != nil {
		return err
	}
	_, _ = w.Write(data)

	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

// 同步记录到Kafka
func (c *Client) backMsgToKafka(data []byte) {
	var msg Msg
	if len(data) == 0 {
		return
	}
	packet := PayloadBytes(data)
	err := packet.UnPacket(&msg)
	if err != nil {
		packet.Release()
		return
	}
	//目前只考虑群聊,进入聊天室等消息入库
	if utils.ContainsBaseType(
		[]uint32{mdata.MsgIDTypeBroadcastRoom, mdata.MsgIDTypeBroadcastJoin, mdata.MsgIDTypeShareBetRecord}, msg.MsgId) {
		c.RoomPushMethod(PostHandleKafkaMsg(&msg).Bytes(), config.GetKafkaTopic().ChatMsgBackTopic)
	}
}

// SyncMember 如果有更新，还需要同步用户信息
func (c *Client) SyncMember(m *mdata.MemberInfo) {
	c.mux.Lock()
	c.member = m
	c.key = m.Name
	c.mux.Unlock()
}

func (c *Client) GetHub() *Hub {
	return c.hub
}

func (c *Client) CheckSelf(username string) bool {
	if c.member == nil {
		return false
	}
	return c.member.Name == username
}

func (c *Client) CheckClientType(clientTypes []string) bool {
	if c.member == nil {
		return false
	}

	for _, clientType := range clientTypes {
		if strings.EqualFold(clientType, c.clientType) {
			return true
		}
	}

	return false
}

func (c *Client) CheckAgent(isAgent string) bool {
	if c.member == nil {
		return false
	}

	if len(isAgent) == 0 {
		return true
	}

	a, _ := strconv.Atoi(isAgent)
	b, _ := strconv.Atoi(c.IsAgent())

	if a == 1 {
		return b == 1
	}

	if a == 0 {
		return b != 1
	}

	return false
}
