package ws

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"msgPushSite/internal/glog"
	"msgPushSite/lib/randid"
	"msgPushSite/mdata"
	"msgPushSite/utils"
)

const (
	SocketMaxConnectSize = 1 * 1024 * 1024
	SocketMaxMsgSize     = 1 * 1024 * 1024
	MaxMessageSize       = 1024 * 12
)

// App hub
type App struct {
	hubs               []*Hub
	roundNext          uint32
	hubSize            int
	wg                 *sync.WaitGroup
	mutex              sync.RWMutex
	ctx                context.Context
	cancel             context.CancelFunc
	RoomPushMethod     PushToKafka
	quit               int32
	onConnectionCreate func(client *Client)
	onConnectionStop   func(client *Client)
	status             bool // 全局房间停用
}

var (
	app *App
)

// InitApp 初始化 APP, 以及启动 hub
func InitApp(push PushToKafka) {
	MsgChan = make(chan *Packet, MsgChanSize)
	numberOfHubs := runtime.NumCPU() * 2
	app = &App{
		wg:             new(sync.WaitGroup),
		hubSize:        numberOfHubs,
		hubs:           make([]*Hub, numberOfHubs),
		RoomPushMethod: push,
	}
	app.ctx, app.cancel = context.WithCancel(context.Background())
	for i := 0; i < numberOfHubs; i++ {
		app.wg.Add(1)
		atomic.AddInt32(&app.quit, 1)
		app.hubs[i] = newHub(app.wg, app.ctx, &app.quit, i)
		go app.hubs[i].run()
	}
	app.wg.Wait()
	go msgRun()
	glog.Infof("Starting %v ws hubs\n", numberOfHubs)

	app.checkExpiredRoom()
	app.checkTimeout()
}

// GetApp 获取当前的应用实例
func GetApp() *App {
	return app
}

// TotalWebsocketConnections 获得连接的总量
func (a *App) TotalWebsocketConnections() int {
	count := int64(0)
	for i := range a.hubs {
		count = count + atomic.LoadInt64(&a.hubs[i].connectCount)
	}

	return int(count)
}

func (a *App) Stop() {
	a.cancel()
	for a.quit != 0 {
		time.Sleep(time.Millisecond * 300)
	}
	for _, h := range a.hubs {
		h.Quit()
	}
	glog.Info("Ws APP管理已安全退出！！")
}

func (a *App) SetOnConnectionCreate(f func(client *Client)) {
	a.onConnectionCreate = f
}

func (a *App) SetOnConnectionStop(f func(client *Client)) {
	a.onConnectionStop = f
}

func (a *App) GetClientsByRidAndUsername(rid, username string) []string {
	res := make([]string, 0)
	for _, hub := range a.hubs {
		ele, ok := hub.GetClientsByRoomId(rid, username)
		if !ok {
			continue
		}
		res = append(res, ele...)
	}
	return res
}

func (a *App) GetClientsByUsername(username string) []string {
	res := make([]string, 0)
	for _, hub := range a.hubs {
		ele := hub.GetClientsByUsername(username)
		if len(ele) == 0 {
			continue
		}
		res = append(res, ele...)
	}
	return res
}

func (a *App) GetClientsByCondition(username string, clientTypes []string, isAgent string) []string {
	res := make([]string, 0)
	for _, hub := range a.hubs {
		ele := hub.GetClientsByCondition(username, clientTypes, isAgent)
		if len(ele) == 0 {
			continue
		}
		res = append(res, ele...)
	}
	return res
}

// HubStop 停掉所有的 hub
//func (a *App) HubStop() {
//	for i := range a.hubs {
//		a.hubs[i].Stop()
//	}
//
//	a.hubs = []*Hub{}
//}

// AllHubStop 停掉所有的hub
func AllHubStop() {
	app.Stop()
}

// 获取最有效的hub
func (a *App) getAvailableHub() (*Hub, error) {

	if a.TotalWebsocketConnections() > SocketMaxConnectSize {
		return nil, errors.New("overflow hub max size")
	}
	return a.next(), nil
}

func (a *App) next() *Hub {
	var n = atomic.AddUint32(&a.roundNext, 1)
	return a.hubs[(int(n)-1)%a.hubSize]
}

// joinRoom 所有房间切换（或进入新房间）必须走这块逻辑，要不然会存在线程不安全问题，比如同一个Room分布在不同的Hub等。
func (a *App) joinRoom(c *Client, roomID, startAt string) {
	var (
		ok   bool
		hub  *Hub
		room *Room
	)
	//房间号需要区分站点
	roomID = c.siteId + "_" + roomID
	a.mutex.Lock()
	defer a.mutex.Unlock()
	// 1. 获取房间所在的Hub和Room TODO 可以改成hash函数取模获取房间对应的hub
	for i := range app.hubs {
		hub = app.hubs[i]
		room, ok = hub.GetRoom(roomID)
		if ok {
			break
		}
		hub = nil
		room = nil
	}
	// 删除老房间的会员ID
	oldRoom := c.room
	if oldRoom != nil {
		oldRoom.Remove(c.Id)
	}
	// 2. 如果新房间不存在，则创建
	if room == nil {
		// 1). 创建房间
		room = NewRoom(roomID, startAt)
		// 2). 将当前连接添加进房间
		room.Add(c)
		// 3). 连接绑定房间
		c.SetRoom(room)
		// 4). 将房间添加进Hub
		c.hub.AddRoom(room)
		return
	}
	// 3. 如果已存在则切换，将新房间ID进行关联
	room.Add(c)
	c.SetRoom(room)
	if c.hub == hub {
		return
	}
	// 3). 删除旧的hub关联
	oldHub := c.hub
	if oldHub != nil {
		oldHub.Remove(c.Id)
	}
	// 4). 更新新的hub
	hub.Add(c)
	c.hub = hub
}

// checkTimeout 巡检建立连接，但是未进行登录的连接close。
func (a *App) checkTimeout() {
	//2分钟检测一次
	mdata.TimingWheel.ScheduleFunc(&mdata.RotateScheduler{Interval: time.Duration(120) * time.Second}, func() {

		const batchSize = 100
		var (
			count int
		)

		for _, hub := range a.hubs {
			if hub == nil {
				continue
			}

			// 遍历 hub 中所有在线用户
			for _, client := range hub.clients {
				user := client.member

				if user != nil {

				}
				count++
				if count >= batchSize {
					//flushLogs()
				}
			}

			// 正常执行超时检测
			//hub.checkTimeout(time.Now().UnixMilli(), traceID)
		}

		// 打印剩余不足 500 的
		//glog.Infof("检测未登录用户结束 ===>> Trace ID:[%s] 耗时:[%.2f秒]", traceID, time.Since(now).Seconds())
	})
}

func (a *App) GetClient(name string, siteId string) *Client {
	for _, hub := range a.hubs {
		if hub == nil {
			continue
		}

		for _, client := range hub.clients {
			user := client.member

			if user != nil && user.Name == name && client.siteId == siteId {
				return client
			}
		}
	}

	return nil
}

// GetAllClient 获取所有的在线用户
func (a *App) GetAllClient() []*Client {
	var out []*Client
	for _, hub := range a.hubs {
		if hub == nil {
			continue
		}
		for _, client := range hub.clients {
			if client != nil && client.member != nil {
				out = append(out, client)
			}
		}
	}
	return out
}

// Paginate 分页处理
func (a *App) Paginate(clients []*Client, page, pageSize int) []*Client {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start >= len(clients) {
		return []*Client{}
	}
	end := start + pageSize
	if end > len(clients) {
		end = len(clients)
	}
	return clients[start:end]
}

// GetClientsByRoomHub 根据hub 获取到在线数据
func (a *App) GetClientsByRoomHub(hubIndex int) []*Client {
	var out []*Client
	for _, hub := range a.hubs {
		if hub == nil {
			continue
		}

		if hub.id == hubIndex {
			for _, client := range hub.clients {
				if client != nil {
					out = append(out, client)
				}
			}
			return out
		}

	}
	return out
}

func (a *App) sendAllClient(schema *BroadcastSchema) {
	traceID := randid.GenerateId()
	glog.Infof("[TraceID:%s] sendAllClient 开始，hub数量: %d", traceID, len(a.hubs))

	//var (
	//	hubIDs    []string
	//	userNames []string
	//	mu        sync.Mutex // 保护 userNames 并发写入
	//	wg        sync.WaitGroup
	//)

	index := -1
	for _, hub := range a.hubs {
		index = index + 1
		if hub == nil {
			continue
		}

		oHub := a.hubs[index]
		//hubIDs = append(hubIDs, fmt.Sprintf("%v", hub.id))
		//
		//wg.Add(1)
		//go func(h *Hub) {
		//	defer wg.Done()
		//
		//	// 收集用户名
		//	var localNames []string
		//	for _, client := range h.clients {
		//		if client != nil && client.member != nil && client.member.Name != "" {
		//			name := client.member.Name
		//			if name != "" {
		//				localNames = append(localNames, name)
		//			}
		//		}
		//
		//	}
		//
		//	// 合并到主列表（加锁）
		//	if len(localNames) > 0 {
		//		mu.Lock()
		//		userNames = append(userNames, localNames...)
		//		mu.Unlock()
		//	}
		//
		//}(hub)

		// 执行广播
		oHub.Broadcast2Global(schema)
	}

	//wg.Wait()
	//
	//glog.Infof("[TraceID:%s] sendAllClient 完成 | hubID列表: [%s]", traceID, strings.Join(hubIDs, ", "))
	//if len(userNames) > 0 {
	//	glog.Infof("[TraceID:%s] 在线用户名列表（共 %d 人）: [%s]", traceID, len(userNames), strings.Join(userNames, ", "))
	//} else {
	//	glog.Infof("[TraceID:%s] 没有在线用户", traceID)
	//}
}

func (a *App) SendAllClientForMsg(msg []byte) {
	traceID := randid.GenerateId()
	glog.Infof("[TraceID:%s] sendAllClientForMsg 开始，hub数量: %d", traceID, len(a.hubs))
	index := -1
	for _, hub := range a.hubs {
		index = index + 1
		if hub == nil {
			continue
		}

		oHub := a.hubs[index]

		for _, client := range oHub.clients {
			if client != nil && client.member != nil {
				client.send <- msg
			}
		}
	}

}

// checkExpiredRoom 赛事存活时间最长5个小时  //
func (a *App) checkExpiredRoom() {
	mdata.TimingWheel.ScheduleFunc(&mdata.RotateScheduler{Interval: time.Minute * 60 * 5}, func() {
		now := time.Now()
		traceID := randid.GenerateId()
		glog.Infof("开始检测赛事超时===>>执行ID:[%s] 开始时间:[%s]", traceID, now.Format(utils.TimeBarFormat))
		for index, _ := range a.hubs {
			hub := a.hubs[index]
			if hub != nil {
				hub.clearRoomInfo(now.Unix())
			}
		}
		glog.Infof("结束检测赛事超时===>>执行ID:[%s] 耗时时间:[%.4f s]", traceID, time.Since(now).Seconds())
	})
}

func JoinRoom(c *Client, roomID, startAt string) {
	app.joinRoom(c, roomID, startAt)
}

func SetConnectionCreate(f func(client *Client)) {
	app.SetOnConnectionCreate(f)
}

func SetConnectionStop(f func(client *Client)) {
	app.SetOnConnectionStop(f)
}

func Pulse() *mdata.PulseInfo {

	lens := len(app.hubs)
	rooms := make([]interface{}, 0)
	for _, hub := range app.hubs {
		for _, value := range hub.rooms {
			var r = new(mdata.Room)
			r.Name = value.GetID()
			r.Clients = len(value.conn)
			r.StartDate = value.startDate.Format(utils.TimeBarFormat)
			r.Created = value.createAt.Format(utils.TimeBarFormat)
			rooms = append(rooms, r)
		}
	}
	return &mdata.PulseInfo{
		Hub:   lens,
		Rooms: rooms,
	}

}
