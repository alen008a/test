package ws

import (
	"strings"
	"sync"
	"time"

	"msgPushSite/internal/glog"
	"msgPushSite/utils"
)

type Room struct {
	id        string             // 房间ID（与站点ID,赛事ID关联）
	conn      map[string]*Client // 房间连接
	mu        sync.RWMutex       // 房间锁
	createAt  time.Time          // 房间创建时间
	status    int32              // 房间状态：0 停用 1 启用
	startDate time.Time          // 房间赛事开始时间
}

func NewRoom(rid, startAt string) *Room {
	startDate, err := utils.ParseTime(startAt)
	if err != nil {
		glog.Errorf("RID[%s] 创建房间开赛时间错误:【%s】| error: %s", rid, startAt, err.Error())
	}
	return &Room{
		id:        rid,
		conn:      make(map[string]*Client),
		createAt:  time.Now(),
		startDate: startDate,
	}
}

func (r *Room) GetID() string {
	return r.id
}

// GetPureID 实际存储的房间ID,前面加入的站点ID，需要截取掉
func (r *Room) GetPureID() string {
	if strings.Count(r.id, "_") >= 2 {
		return r.id[strings.Index(r.id, "_")+1:]
	}
	return r.id
}

func (r *Room) Add(c *Client) {
	r.mu.Lock()
	r.conn[c.Id] = c
	r.mu.Unlock()
}

func (r *Room) Broadcast(schema *BroadcastSchema) {
	if schema.Msg.MsgId == 0 {
		return
	}
	r.mu.RLock()
	conns := make([]*Client, 0, len(r.conn))
	for _, c := range r.conn {
		if c != nil && !c.IsClose() {
			conns = append(conns, c)
		}
	}

	r.mu.RUnlock()

	// 可选：只克隆一次，作为“只读、不可复用”的广播帧
	for _, c := range conns {
		if !c.trySend(schema.buff, 50*time.Millisecond) {
			glog.Warnf("room broadcast drop |room=%s key=%s", r.id, c.key)
		}
	}
}

func (r *Room) Remove(key string) {
	r.mu.Lock()
	delete(r.conn, key)
	r.mu.Unlock()
}

func (r *Room) ClearConn() {
	r.mu.Lock()
	for key, c := range r.conn {
		c.Close()
		delete(r.conn, key)
	}
	r.mu.Unlock()
}

func (r *Room) GetConnectionsByKey(username string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]string, 0)
	for _, c := range r.conn {
		if !c.CheckSelf(username) {
			continue
		}
		res = append(res, c.Id)
	}
	return res
}
