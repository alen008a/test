package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"msgPushSite/lib/cache"
	"msgPushSite/lib/randid"
	"msgPushSite/service"
	"msgPushSite/service/sego"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"msgPushSite/db/redisdb/core"
	"msgPushSite/mdata/rediskey"
	"msgPushSite/utils"

	"msgPushSite/internal/glog"
	"msgPushSite/mdata"

	"github.com/panjf2000/ants/v2"
)

const (
	MsgChanSize        = 4096 * 10 // msg chan 的队列大小
	BroadcastQueenSize = 1024 * 4  // 广播chan队列大小
	RegisterChanSize   = 256 * 2   // 注册人数队列大小
	TokenTTL           = 8 * time.Hour
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.

type Key = string
type ClientType = string

type Hub struct {
	id           int
	wg           *sync.WaitGroup
	mutex        *sync.RWMutex         // 锁
	connectCount int64                 // 连接数量
	rooms        map[string]*Room      // 房间
	clients      map[string]*Client    // 连接
	broadcast    chan *BroadcastSchema // 消息广播
	//register     chan *Client          // Register requests from the clients.
	unregister   chan *Client    // Unregister requests from clients.
	ctx          context.Context // 上线文管理，Hub生命周期与APP绑定。所以要与APP同时结束
	ExplicitStop bool            // 标识该 hub 是否已经停止, 默认为 false 不停止
	quitPtr      *int32          // 退出引用，用于APP关闭
}

func newHub(wg *sync.WaitGroup, ctx context.Context, quitPtr *int32, id int) *Hub {
	return &Hub{
		id:           id,
		wg:           wg,
		mutex:        new(sync.RWMutex),
		connectCount: 0,
		//register:     make(chan *Client, RegisterChanSize),
		broadcast:    make(chan *BroadcastSchema, BroadcastQueenSize),
		unregister:   make(chan *Client, RegisterChanSize),
		clients:      make(map[string]*Client),
		rooms:        make(map[string]*Room),
		ctx:          ctx,
		ExplicitStop: false,
		quitPtr:      quitPtr,
		//unAuth:       make([]*Client, 0),
	}
}

func (h *Hub) run() {
	h.wg.Done()
	var doStart func()            // 启动 hub
	var doRecoverableStart func() // 抓取 panic 然后重启 hub
	var doRecover func()          // 抓取 panic

	doStart = func() {
		for {
			select {
			//case client := <-h.register:
			//	// 1. 进入房间
			//	JoinRoom(client, client.roomID)
			//	// 2. 激活连接
			//	client.isLogin = true
			//	// 3. 连接数加1
			//	atomic.AddInt64(&h.connectCount, 1)

			case client := <-h.unregister:
				glog.Infof("client release id:[%s] | name:[%s] | ip:[%s] ", client.Id, client.key, client.ip)
				// 1. 释放client资源
				client.release()
				// 2. 连接数减1
				atomic.AddInt64(&h.connectCount, -1)

			case message := <-h.broadcast:
				h.handler(message)
			case <-h.ctx.Done():
				atomic.AddInt32(h.quitPtr, -1)
				h.mutex.Lock()
				// 1. 释放房间连接
				for _, room := range h.rooms {
					room.ClearConn()
				}
				h.mutex.Unlock()

				h.rooms = nil
				// 2. 释放Hub连接，并断开连接
				for key, _ := range h.clients {
					delete(h.clients, key)
					//cli.conn.Close()
				}

				// 3. 关闭 chan stop
				h.ExplicitStop = true

				// 将是否停止设置为 true
				return
			}
		}
	}

	doRecover = func() {
		if !h.ExplicitStop {
			if r := recover(); r != nil {
				glog.Errorf("hub Recovering from Hub panic. Panic was: %v", r)
			} else {
				glog.Error("hub stopped unexpectedly. Recovering.")
			}
			glog.Emergency("ws hub panic |stack=%s", string(debug.Stack()))
			go doRecoverableStart()
		}
	}

	doRecoverableStart = func() {
		defer doRecover()
		doStart()
	}

	go doRecoverableStart()
}

// 广播消息处理
func (h *Hub) handler(schema *BroadcastSchema) {
	switch schema.Msg.MsgFlag {
	case MsgFlagSelf: // 单播

		h.Broadcast2Self(schema)
	case MsgFlagRoom: //针对房间号推送

		if schema.Msg.MsgId == mdata.MsgIDMatchTerminated {
			h.clearRoomById(schema)
		} else {
			h.Broadcast2Room(schema)
		}
	case MsgFlagGlobal: // 全局广播

		// 如果是红包雨 者需要处理所有的hub
		h.Broadcast2Global(schema)
		// 需要处理多节点的情况

	case MsgFlagConditionGlobal: // 全局条件广播
		h.Broadcast2ConditionGlobal(schema)
	}
}

func (h *Hub) GetRoom(rid string) (res *Room, ok bool) {
	h.mutex.RLock()
	res, ok = h.rooms[rid]
	h.mutex.RUnlock()
	return
}

func (h *Hub) AddRoom(room *Room) {
	if h == nil {
		glog.Infof("加入房间时，Hub为nil")
		return
	}
	if h.rooms == nil {
		glog.Infof("加入房间时，Hub的rooms为nil")
		h.rooms = make(map[string]*Room, 0)
		return
	}
	h.mutex.Lock()
	h.rooms[room.id] = room
	h.mutex.Unlock()
}

func (h *Hub) Add(cli *Client) {
	h.mutex.Lock()
	h.clients[cli.Id] = cli
	h.SaveMemberToRedis(cli)
	h.mutex.Unlock()
}

func (h *Hub) Remove(id string) {

	h.mutex.Lock()
	cli, _ := h.clients[id]
	if cli != nil {
		h.RmMemberFromRedis(cli)
	}
	delete(h.clients, id)
	h.mutex.Unlock()
}

func (h *Hub) Quit() {
	keys, err := core.ScanKeys(rediskey.ClientOnlineSetPrefix, 1000)
	if err != nil {
		glog.Infof("Hub QuitDelKey core.ScanKeys error:%+v key:%s", err, rediskey.ClientOnlineSetPrefix)
	}
	if len(keys) > 0 {
		core.DelKey(keys...)
	}
	keys, err = core.ScanKeys(rediskey.ClientOnlineClientSetPrefix, 1000)
	if err != nil {
		glog.Infof("Hub QuitDelKey core.ScanKeys error:%+v key:%s", err, rediskey.ClientOnlineClientSetPrefix)
	}
	if len(keys) > 0 {
		core.DelKey(keys...)
	}
	glog.Infof("Hub QuitDelKey")
}

func (h *Hub) SaveMemberToRedis(cli *Client) {
	member := cli.key
	if len(member) == 0 {
		return
	}
	score, _ := core.ZScore(fmt.Sprintf(rediskey.ClientOnlineSet, cli.siteId), member)
	cliType := rediskey.GetClientType(strings.ToLower(cli.clientType))
	core.ZAdd(fmt.Sprintf(rediskey.ClientOnlineSet, cli.siteId), member, float64(cliType.Score|int(score)))

	key := fmt.Sprintf(rediskey.ClientOnlineClientSet, cli.siteId, cliType.Type)
	core.SAdd(key, member)
	glog.Debugf("SaveMemberToRedis key: %s, member: %s, score: %v", fmt.Sprintf(rediskey.ClientOnlineSet, cli.siteId), member, float64(cliType.Score|int(score)))
}

func (h *Hub) RmMemberFromRedis(cli *Client) {
	member := cli.key
	core.ZRem(fmt.Sprintf(rediskey.ClientOnlineSet, cli.siteId), member)

	cliType := rediskey.GetClientType(strings.ToLower(cli.clientType))
	key := fmt.Sprintf(rediskey.ClientOnlineClientSet, cli.siteId, cliType.Type)
	core.SRem(key, strings.ToLower(cli.clientType), member)
	glog.Debugf("RmMemberFromRedis key: %s, member: %s", fmt.Sprintf(rediskey.ClientOnlineSet, cli.siteId), member)
}

func (h *Hub) GetClient(cid string) (cli *Client, ok bool) {
	h.mutex.RLock()
	cli, ok = h.clients[cid]
	h.mutex.RUnlock()
	return

}

func (h *Hub) PrintClient() {
	h.mutex.RLock()
	list := make([]string, 0)
	for _, c := range h.clients {
		list = append(list, c.Id)
	}
	glog.Infof("$$$$$$$$$$$$$$-----------  HUB[%d] PrintClient :%+v", h.id, list)
	h.mutex.RUnlock()
}

func (h *Hub) PrintRoom() {
	h.mutex.RLock()
	list := make([]string, 0)
	for _, c := range h.rooms {
		list = append(list, c.id)
	}
	glog.Infof("$$$$$$$$$$$$$$----------- HUB[%d] PrintRoom :%+v", h.id, list)
	h.mutex.RUnlock()
}

func (h *Hub) GetContext() context.Context {
	return h.ctx
}

// Broadcast2Room 针对房间发送
func (h *Hub) Broadcast2Room(schema *BroadcastSchema) {
	h.mutex.RLock()
	room, ok := h.rooms[schema.Msg.RoomId]
	h.mutex.RUnlock()
	if !ok {
		return
	}
	room.Broadcast(schema)
}

func (h *Hub) GetHubID() int {
	return h.id
}

// Broadcast2Self 单播
func (h *Hub) Broadcast2Self(schema *BroadcastSchema) {
	if schema == nil {
		return
	}
	if schema.Self == nil {
		return
	}

	for _, value := range schema.Self {
		cli, ok := h.GetClient(value)
		if !ok {
			continue
		}
		send(cli, schema)
	}
}

func containsVipLevel(vipLevel string, grade string) bool {
	levels := strings.Split(vipLevel, ",") // 按逗号分隔
	for _, v := range levels {
		if strings.TrimSpace(v) == grade {
			return true
		}
	}
	return false
}

// Broadcast2Global 全局广播
func (h *Hub) Broadcast2Global(schema *BroadcastSchema) {
	traceID := randid.GenerateId()

	// 收集所有用户名
	usernames := make([]string, 0, len(h.clients))
	for _, v := range h.clients {
		if v.member != nil {
			usernames = append(usernames, v.member.Name)
		}

	}

	glog.Infof("[TraceID:%s] Broadcast2Global 开始 | hub:%v | clientCount:%v | usernames:%v",
		traceID, h.id, len(h.clients), usernames)

	for _, v := range h.clients {
		func(client *Client) {
			defer func() {
				if r := recover(); r != nil {
					glog.Errorf("[TraceID:%s] panic recovered in Broadcast2Global | hub:%v | client:%v(username:%s) | err: %v\n%s",
						traceID, h.id, client, client.member.Name, r, debug.Stack())
				}
			}()
			h.safeBroadcast(client, schema, traceID)
		}(v)
	}

	glog.Infof("[TraceID:%s] Broadcast2Global 结束 | hub:%v", traceID, h.id)
}

// 封装安全广播逻辑，避免重复代码 + 提高清晰度
func (h *Hub) safeBroadcast(v *Client, schema *BroadcastSchema, traceID string) {
	if v == nil || v.member == nil {
		glog.Infof("[TraceID:%s] client 或 member 为 nil，跳过广播 | hub:%v", traceID, h.id)
		return
	}

	memberName := v.member.Name
	siteID := v.siteId
	logPrefix := fmt.Sprintf("[TraceID:%s] [hub:%v] [用户:%s] [siteId:%s]", traceID, h.id, memberName, siteID)

	// 普通消息
	if schema.Msg.MsgId != mdata.MsgIdRedPackageRain {
		send(v, schema)
		glog.Infof("%s 其他非红包雨消息 → 发送成功", logPrefix)
		return
	}

	// 红包雨消息处理
	dataMap, ok := schema.Msg.MsgData.(map[string]interface{})
	if !ok {
		glog.Infof("%s 失败：MsgData 格式错误", logPrefix)
		return
	}

	var envelopeMsg ResEnvelopeMsgVo
	raw, _ := json.Marshal(dataMap)
	if err := json.Unmarshal(raw, &envelopeMsg); err != nil {
		glog.Infof("%s 失败：数据解析失败", logPrefix)
		return
	}

	if strconv.Itoa(int(envelopeMsg.SiteId)) != siteID {
		glog.Infof("%s 跳过：SiteId 不匹配", logPrefix)
		return
	}

	// VIP 限制
	if envelopeMsg.Type == 1 {

		redis := rediskey.NewRedEnvelopeHashRedis(envelopeMsg.RedPackId, int(envelopeMsg.SiteId))
		envelope := redis.GetActivityEnvelopeForMessage(h.GetContext(), envelopeMsg.RedPackId, envelopeMsg.SiteId)
		if envelope != nil && envelope.Type == 1 {
			if !containsVipLevel(envelope.VipLevel, strconv.Itoa(v.member.Vip)) {
				glog.Infof("%s 跳过：VIP 不匹配", logPrefix)
				return
			}
		} else {
			glog.Infof("GetActivityEnvelopeForMessage 获取红包为空:%#v", envelopeMsg)
		}
	}

	// 指定用户
	if envelopeMsg.Type == 0 {

		key := fmt.Sprintf(rediskey.CurrentRedEnvelopeUserListKey, envelopeMsg.SiteId, envelopeMsg.RedPackId)
		existed, err := core.SIsMember(key, memberName)
		if err != nil {
			glog.Infof("%s 失败：Redis 错误 %v", logPrefix, err)
			return
		}
		if !existed {
			glog.Infof("%s 跳过：不在指定用户列表中", logPrefix)
			return
		}
	}

	// 满足条件后发送
	send(v, schema)
	glog.Infof("%s 红包雨消息 → 发送成功  data :%v", logPrefix, envelopeMsg)
}

// Broadcast2ConditionGlobal 全局条件广播
func (h *Hub) Broadcast2ConditionGlobal(schema *BroadcastSchema) {
	for _, v := range h.clients {
		if !v.CheckClientType(schema.Msg.ClientTypes) {
			continue
		}

		if !v.CheckAgent(schema.Msg.IsAgent) {
			continue
		}

		send(v, schema)
		//sendV2(v, schema)
	}
}

// 依赖：utils.Clone，cli.trySend（你已有实现：Clone + select + time.After）
func send(cli *Client, schema *BroadcastSchema) {

	// 2) 短超时入队（避免阻塞）
	const enqueueWait = 50 * time.Millisecond
	ok := cli.trySendNoClone(schema.buff, enqueueWait)
	if !ok {
		glog.Warnf("send drop |key=%s |room=%s |msgId=%d", cli.key, cli.GetRoom().GetID(), schema.Msg.MsgId)
		return
	}

	// 3) 只有入队成功再做后续副作用
	if schema.Msg.MsgId == mdata.MsgIDInternalNotice {
		seq := schema.Msg.Seq
		if len(seq) != 0 {
			// 计数
			key := fmt.Sprintf(rediskey.ClientSendCount, cli.siteId, seq)
			core.Incr(key)
		}

		name := schema.Msg.Key
		if name != "" {
			// 清理通知集合与记录
			zsetKey := fmt.Sprintf(rediskey.ClientNoticeSet, cli.siteId, name, cli.clientType)
			member := fmt.Sprintf(rediskey.ClientNoticeKey, cli.siteId, schema.Msg.Seq, name)
			// 建议：用 Pipeline 批量
			core.ZRem(zsetKey, member)
			core.DelKey(member)
			glog.Debugf("异步历史消息清理 |name=%s |zset=%s |member=%s", name, zsetKey, member)
		} else {
			hashKey := fmt.Sprintf(rediskey.ClientNoticeRecordHash, cli.siteId, schema.Msg.Seq, cli.clientType)
			core.HSet(hashKey, cli.GetUsername(), "1")
		}
	}
}

// checkTimeout 未登录用户扫描
func (h *Hub) checkTimeout(start int64, traceID string) {

	h.mutex.Lock()
	for index, _ := range h.clients {
		cli := h.clients[index]
		if cli.isLogin {
			continue
		}
		// 如果连接创建时间大于30分钟未登录，则踢出连接 TODO 将来改成动态配置
		if start-cli.creatAt < 30*60 {
			continue
		}
		//cli.Close()
		//glog.Infof("connect un auth traceIuD[%s] ID:[%s] IP:[%s] clientType:[%s]", traceID, cli.Id, cli.Ip(), cli.ClientType())
	}
	h.mutex.Unlock()
}

// clearRoomInfo 定时清除赛事已经过期的房间
func (h *Hub) clearRoomInfo(start int64) {
	h.mutex.Lock()
	for key, value := range h.rooms {
		if value == nil {
			continue
		}
		// 1. 判断开赛时间是否大于15天，未到15天则跳过
		if start-value.startDate.Unix() < 60*60*24*15 {
			continue
		}
		glog.Infof("start clear already expired room ===>> RoomID:【%s】CreateAt:【%s】LiveDate:【%s】Online:【%d】",
			value.id,
			value.createAt.Format(utils.TimeBarFormat),
			value.startDate.Format(utils.TimeBarFormat),
			len(value.conn),
		)
		// 2. 清除房间所有连接
		value.ClearConn()
		delete(h.rooms, key)
	}
	h.mutex.Unlock()
}

// clearRoomInfo 定时清除赛事已经过期的房间
func (h *Hub) clearRoomById(schema *BroadcastSchema) {

	h.mutex.Lock()
	defer h.mutex.Unlock()
	value := h.rooms[schema.Msg.RoomId]
	if value == nil {
		return
	}
	glog.Infof("start clear already expired room ===>> RoomID:【%s】CreateAt:【%s】LiveDate:【%s】Online:【%d】",
		value.id,
		value.createAt.Format(utils.TimeBarFormat),
		value.startDate.Format(utils.TimeBarFormat),
		len(value.conn),
	)
	value.ClearConn()
	delete(h.rooms, schema.Msg.RoomId)

}

// msgRun 接受外部消息发送到每一个端
func msgRun() {
	// 设置协程池的大小为 cpu 的核数
	//poolSize := runtime.NumCPU()
	//if poolSize < 100 {
	//	poolSize = 100
	//} else if poolSize > 500 {
	//	poolSize = 500
	//}
	poolSize := 1024 * 8

	pool, err := ants.NewPool(poolSize)
	if err != nil {
		glog.Emergency("ws hub ants NewPool error|err=>%v", err)
		return
	}
	defer pool.Release()

	for {
		select {
		case message := <-MsgChan:
			var (
				msg, msgN Msg
				broad     = new(BroadcastSchema)
			)
			// 1. 判断是不是空数据
			if message.Len() == 0 {
				break
			}
			// 2. 进行序列化，获取msg
			err := message.UnPacket(&msg)
			if err != nil {
				glog.Errorf("msgRun err=%v |data=%s", err, message.String())
				break
			}
			glog.Infof("收到消息 msg: %s", mdata.MustMarshal(msg))
			broad.Msg = &msg
			msgN = msg
			if strings.Count(msgN.RoomId, "_") >= 2 {
				msgN.RoomId = msgN.RoomId[strings.Index(msgN.RoomId, "_")+1:]
			}
			if msg.MsgId == mdata.MsgIdRedPackageRain {
				glog.Infof("红包雨消息:%s", string(mdata.MustMarshal(msg)))
			}

			if msg.MsgId == mdata.MsgIdRedPackageReceive {
				//glog.Infof("接收到获取红包消息%v", string(mdata.MustMarshal(msg)))
			}

			//禁言类消息清楚本地缓存
			if mdata.MsgIDTypeSpeechStatus == msg.MsgId {
				cache.DeleteCache(fmt.Sprintf(rediskey.MemberSpeechStatus, msg.SiteId))
				cache.DeleteCache(fmt.Sprintf(rediskey.MemberSpeechBannedDuration, msg.SiteId))
			}

			// 3.单播（后台类 消息仍需找到client发送，敏感词把房间消息变成单播的已推送过client 无需再匹配client发送）
			if msg.MsgFlag == MsgFlagSelf {
				if msg.MsgId == mdata.MsgIDTypeBroadcastRoom || msg.MsgId == mdata.MsgIDTypeShareBetRecord {
					//敏感词把房间消息变成单播的已推送client， 无需再匹配client发送，直接退回
					break
				} else if msg.MsgId == mdata.MsgIDInternalNotice {
					broad.Self = app.GetClientsByCondition(msg.Key, msg.ClientTypes, msg.IsAgent)
				} else {
					broad.Self = app.GetClientsByUsername(msg.Key)
				}
			}

			//为防止出现屏蔽消息后， 再次刷新进入聊天室时， 被屏蔽的消息会再次显示， 所以这里额外再处理从redis中删除屏蔽的消息
			if msg.MsgId == mdata.MsgIDTypeMsgShield {
				err = delBannedMsgInCache(msg)
				if err != nil {
					glog.Errorf("ws.msgRun DelBannedMsgInCache err|message=>%v,err=>%v", string(mdata.MustMarshal(msg)), err)
				}
			}
			message.Reset()
			message.Packet(&msgN)

			// 解决内存可能会复用问题
			frame := utils.Clone(message.Bytes()) // bytes.Clone 或 append(nil, ...) 都可
			broad.buff = frame

			message.Release()
			for i := range app.hubs {
				tHub := app.hubs[i]

				err1 := pool.Submit(
					func() {
						var timeoutChan = make(chan struct{})
						var tw = mdata.TimingWheel.AfterFunc(time.Second*5, func() { timeoutChan <- struct{}{} })

						select {
						case tHub.broadcast <- broad:
						case <-timeoutChan: // 增加一个超时机制
							glog.Errorf("broadcastTimeout message: %v", broad)
						}

						tw.Stop()
					},
				)
				if err1 != nil {
					glog.Errorf("submit data=%+v err: %v", broad, err)
				}
			}
		case <-app.ctx.Done():
			return
		}
	}
}

func (h *Hub) GetClientsByRoomId(rid, username string) ([]string, bool) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	room, ok := h.rooms[rid]
	if !ok {
		return nil, false
	}
	res := room.GetConnectionsByKey(username)
	if len(res) == 0 {
		return nil, false
	}
	return res, true
}

func (h *Hub) GetClientsByUsername(username string) []string {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	res := make([]string, 0)
	for _, c := range h.clients {
		if !c.CheckSelf(username) {
			continue
		}
		res = append(res, c.Id)
	}
	return res
}

func (h *Hub) GetClientsByCondition(username string, clientTypes []string, isAgent string) []string {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	res := make([]string, 0)
	for _, c := range h.clients {
		if !c.CheckSelf(username) {
			continue
		}

		if !c.CheckClientType(clientTypes) {
			continue
		}

		if !c.CheckAgent(isAgent) {
			continue
		}

		res = append(res, c.Id)
	}
	return res
}

// 删除redis中被屏蔽的消息
func delBannedMsgInCache(msg Msg) error {

	glog.Infof("进入delBannedMsgInCache|msg=>%#v", msg)

	if len(msg.Seq) < 1 || len(msg.RoomId) < 1 {
		return errors.New(fmt.Sprintf("invalid Seq=>%v or RoomId=>%v", msg.Seq, msg.RoomId))
	}

	if strings.Count(msg.RoomId, "_") >= 2 {
		msg.RoomId = msg.RoomId[strings.Index(msg.RoomId, "_")+1:]
	}
	redisKey := fmt.Sprintf(rediskey.LiveMatchMessage, msg.SiteId, msg.RoomId)
	cacheMsgLen, err := core.LLen(redisKey, false)
	if err != nil || cacheMsgLen < 1 {
		return errors.New(fmt.Sprintf("memer.DelBannedMsgInCache get room redis msgLen failed|redis key=>%v,msgLen=>%v, err=>%v", redisKey, cacheMsgLen, err))
	}

	//批量获取redis中聊天室的消息数据， 找到被屏蔽的消息， 执行删除
	batchNum := 200                                                   //从redis中每批获取的数量
	batch := int(math.Ceil(float64(cacheMsgLen) / float64(batchNum))) //总共需要执行几次

	for i := 1; i <= batch; i++ {
		start := (i - 1) * batchNum //每一批查询时，redis list中的起始index
		end := start + batchNum - 1 //每一批查询时，redis list中的截止index

		cacheData, err := core.LRange(false, redisKey, int64(start), int64(end))
		if err != nil && err != core.RedisNil {
			glog.Errorf("memer.DelBannedMsgInCache get room redis msgData failed|redis key=>%v,start=>%v, end=>%v, err=>%v", redisKey, start, end, err)
			continue
		}
		if len(cacheData) < 1 {
			glog.Errorf("memer.DelBannedMsgInCache get room redis msgData no data|redis key=>%v, start=>%v,end=>%v", redisKey, start, end)
			continue
		}

		//当前批次拿到数据， 以seq找出对应的消息，执行删除
		for j := range cacheData {
			tmpMsg := new(mdata.BroadcastRoomRspSchema)
			err = mdata.Cjson.Unmarshal([]byte(cacheData[j]), tmpMsg)
			if err != nil {
				glog.Errorf("memer.DelBannedMsgInCache decode room redis msgData err|redis key=>%v, cacheData=>%v,err=>%v", redisKey, cacheData[j], err)
				continue
			}
			if len(tmpMsg.Seq) > 0 && tmpMsg.Seq == msg.Seq {
				err = core.LSet(redisKey, int64(j), "  ")
				if err == nil {
					err = core.LRem(redisKey, 0, "  ")
					if err != nil {
						//重新设置被屏蔽的消息后， 删除时出错
						return errors.New(fmt.Sprintf("memer.DelBannedMsgInCache del room redis msgData err|redis key=>%v, cacheData=>%v,err=>%v", redisKey, cacheData[j], err))
					}
					//重新设置后，删除成功，记录一下
					glog.Infof("memer.DelBannedMsgInCache del room redis succeed|redis key=>%v, cacheData=>%v", redisKey, cacheData[j])
					return nil
				} else {
					//重新设置被屏蔽的消息出错
					return errors.New(fmt.Sprintf("memer.DelBannedMsgInCache reset room redis msgData err|redis key=>%v, cacheData=>%v,err=>%v", redisKey, cacheData[j], err))
				}
			}
		}
	}
	return nil
}

// 敏感词处理
func filter(m *Msg) error {
	bs, err := mdata.Cjson.Marshal(m.MsgData)
	if err != nil {
		return err
	}

	msg := mdata.Cjson.Get(bs, "data", "msg").ToString()

	//先去除空格
	msg = strings.ReplaceAll(msg, " ", "")
	//是否存在连续字符
	hsw, _ := sego.Sgmt.Dictionary().IsExistContinuousWord(msg)
	if hsw {
		return mdata.SensitiveRepeatedError
	}
	_, _, hasKeyword := sego.Sgmt.Dictionary().GetNewTrie().Filter(msg)
	if hasKeyword {
		return mdata.SensitiveRepeatedError
	}

	return nil
}

func FilterContent(msg string) (flag int, keyword string) {
	//先去除空格
	msg = strings.ReplaceAll(msg, " ", "")
	//是否存在连续字符
	hsw, sw := sego.Sgmt.Dictionary().IsExistContinuousWord(msg)
	if hsw {
		return 2, sw
	}
	_, swArr, hasKeyword := sego.Sgmt.Dictionary().GetNewTrie().Filter(msg)
	if hasKeyword {
		return 1, strings.Join(swArr, " ")
	}
	return 0, ""
}

// PostHandleKafkaMsg 处理后置消息
func PostHandleKafkaMsg(m *Msg) *mdata.BroadcastRoomKafkaSchema {
	siteId, _ := strconv.Atoi(m.SiteId)
	broadcastRoomRspSchema := &mdata.BroadcastRoomKafkaSchema{
		MsgId:       m.MsgId,
		MsgFlag:     m.MsgFlag,
		SiteId:      siteId,
		Name:        m.Key,
		Seq:         m.Seq,
		Status:      1,
		EsIndexName: m.EsIndexName,
	}
	bs, err := mdata.Cjson.Marshal(m.MsgData)
	if err == nil {
		broadcastRoomRspSchema.Msg = mdata.Cjson.Get(bs, "data", "msg").ToString()
		broadcastRoomRspSchema.VIP = mdata.Cjson.Get(bs, "data", "vip").ToInt()
		broadcastRoomRspSchema.Nickname = mdata.Cjson.Get(bs, "data", "nickname").ToString()
		broadcastRoomRspSchema.MemberId = mdata.Cjson.Get(bs, "data", "memberId").ToInt()
		broadcastRoomRspSchema.Timestamp = mdata.Cjson.Get(bs, "data", "timestamp").ToString()
		broadcastRoomRspSchema.Category = mdata.Cjson.Get(bs, "data", "category").ToInt()
		broadcastRoomRspSchema.CategoryType = mdata.Cjson.Get(bs, "data", "categoryType").ToInt()
		broadcastRoomRspSchema.AllowReport = mdata.Cjson.Get(bs, "data", "allowReport").ToInt()
		broadcastRoomRspSchema.IsReported = mdata.Cjson.Get(bs, "data", "isReported").ToInt()
		if broadcastRoomRspSchema.Category == 1 { //普通聊天才做敏感词校验
			flag, keyWord := FilterContent(broadcastRoomRspSchema.Msg)
			broadcastRoomRspSchema.Flag = int64(flag)
			broadcastRoomRspSchema.SensitiveWord = keyWord
		}
	}
	broadcastRoomRspSchema.CreatedAt = fmt.Sprintf("%s+08:00", strings.Replace(broadcastRoomRspSchema.Timestamp, " ", "T", -1))

	if m.RoomId != "" {
		if strings.Count(m.RoomId, "_") >= 2 {
			m.RoomId = m.RoomId[strings.Index(m.RoomId, "_")+1:]
		}
		broadcastRoomRspSchema.RoomId = m.RoomId
		matchData, _ := service.GetRoomInfo(m.SiteId, m.RoomId)
		//查找赛事信息
		if matchData != nil {
			broadcastRoomRspSchema.MatchId = matchData.MatchId
			broadcastRoomRspSchema.MatchCate = matchData.MatchCate
			broadcastRoomRspSchema.Venue = matchData.Venue
			broadcastRoomRspSchema.Home = matchData.Home
			broadcastRoomRspSchema.Away = matchData.Away
			broadcastRoomRspSchema.League = matchData.League
		}
	}
	return broadcastRoomRspSchema
}

// 预备处理消息
func preHandleMsg(m *Msg) {
	switch {
	case mdata.MsgIDTypeBroadcastRoom == m.MsgId:
		err := filter(m)
		if err != nil {
			glog.Errorf("filter %s %s", m.Seq, err.Error())
			m.MsgFlag = MsgFlagSelf
		}
	case mdata.MsgIDTypeShareBetRecord == m.MsgId:
		m.MsgId = mdata.MsgIDTypeShareBetRecord
		m.MsgFlag = MsgFlagRoom
	//移动端旧包历史信息不兼容新msgid的信息展示，暂时不入库
	case mdata.MsgScorePredictionPush == m.MsgId:
		m.MsgFlag = MsgFlagRoom
	case mdata.MsgIDTypeSpeechStatus == m.MsgId,
		mdata.MsgIDTypeMsgShield == m.MsgId,
		mdata.MsgIDTypeRoomStatus == m.MsgId,
		mdata.MsgIDTypeChatVIPLevel == m.MsgId,
		mdata.MsgIDTypeBroadcastJoin == m.MsgId,
		mdata.MsgIDTypeSelfJoin == m.MsgId,
		mdata.MsgIDTypeAllRoomMaintain == m.MsgId,
		mdata.MsgIDLiveScorePush == m.MsgId,
		mdata.MsgIDMatchChatClear == m.MsgId,
		mdata.MsgIDLiveGiftPush == m.MsgId,
		mdata.MsgIDTypeNoLogin == m.MsgId,
		mdata.MsgIDTypeApiApp == m.MsgId,
		mdata.MsgIDTypeChatHistory == m.MsgId, //history
		mdata.MsgIDMatchTerminated == m.MsgId:
	case m.MsgId > mdata.MsgSimpleMsgMark:
	default:
		glog.Errorf("packetRun unknown type message: %v", m)
	}
}
