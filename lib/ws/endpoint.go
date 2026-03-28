package ws

import (
	"errors"
	"fmt"
	"sync"
)

var gDispatch *Dispatch

// Endpoint messageFlag表示广播的方式，0表示只推送给自己
type Endpoint func(c *Context, req *Packet, rsp *Payload, msg *Msg) (messageFlag MsgFlag)

type Wrapper func(e Endpoint) Endpoint

type Dispatch struct {
	dispatch map[uint32]Endpoint
	lock     *sync.RWMutex
}

func RegisterEndpoint(msgId uint32, endpoint Endpoint) {
	if gDispatch == nil {
		gDispatch = &Dispatch{lock: new(sync.RWMutex), dispatch: make(map[uint32]Endpoint)}
	}

	//去掉加锁，只有服务启动的时候会注册，不存在并发

	if _, ok := gDispatch.dispatch[msgId]; ok {
		panic("endpoint is already register")
	}

	gDispatch.dispatch[msgId] = endpoint
}

func getEndpoint(msgId uint32) (endpoint Endpoint, err error) {
	var ok = false
	if endpoint, ok = gDispatch.dispatch[msgId]; !ok {
		return nil, errors.New(fmt.Sprintf("no endpoint: %v", msgId))
	}
	return
}
