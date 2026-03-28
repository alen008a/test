package ws

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// PushToKafka 推送房间的方法
type PushToKafka func(data []byte, topic string, key ...string)

// MsgFlag 消息广播类型
type MsgFlag = uint32

const (
	MsgFlagSelf            MsgFlag = iota // 单播
	MsgFlagRoom                           // 房间广播
	MsgFlagGlobal                         // 全局广播
	MsgFlagConditionGlobal                // 全局条件广播
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 30 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 3 * 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

// upgrade ws upgrade
var upgrade = websocket.Upgrader{
	ReadBufferSize:  SocketMaxMsgSize,
	WriteBufferSize: SocketMaxMsgSize,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// state 连接状态
type state = uint32

const (
	stateStop state = iota
	statePending
	stateOk
)
