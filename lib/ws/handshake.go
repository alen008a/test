package ws

import (
	ctx "context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"msgPushSite/internal/context"
	"msgPushSite/internal/glog"
	libip "msgPushSite/lib/ip"
	"msgPushSite/lib/randid"
	"msgPushSite/mdata"
	"msgPushSite/utils"
)

func Handshake(c *context.Context, clientType, siteId, xApiXXX string, pushToKafka PushToKafka) error {
	var payload = &Payload{
		StatusCode: utils.StatusOK,
		Message:    "连接成功",
		Data:       nil,
	}

	var sendMessage = &Msg{
		MsgId:   mdata.MsgIdTypeConnect,
		MsgData: payload,
	}

	var err error
	siteIdInt, _ := strconv.Atoi(siteId)
	if siteIdInt < 1 {
		payload.StatusCode = utils.ErrRefuse
		payload.Message = "未知的站点"
		err = errors.New(payload.Message)
		return err
	}
	// TODO 是否打开前端发送请求的X-API-XXX校验
	//if config.GetConfig().MasterOpenSecretKey {
	//	if len(xApiXXX) == 0 {
	//		payload.StatusCode = utils.ErrRefuse
	//		payload.Message = "非法请求"
	//		err = errors.New(payload.Message)
	//		return err
	//	}
	//
	//	//解密校验
	//	result, err := utils.AesCBCPk7DecryptHex(
	//		xApiXXX,
	//		config.GetConfig().BytesMasterSecretKey,
	//		config.GetConfig().BytesMasterIV,
	//	)
	//
	//	if err != nil || len(result) == 0 {
	//		payload.StatusCode = utils.ErrRefuse
	//		payload.Message = "非法请求"
	//		err = errors.New(payload.Message)
	//		return err
	//	}
	//}

	// 获取客户端类型
	if _, ok := utils.ClientTypeMap[clientType]; !ok {
		payload.StatusCode = utils.ErrRefuse
		payload.Message = "客户端类型不存在"
		err = errors.New(payload.Message)
		return err
	}

	hub, err := app.getAvailableHub()
	if err != nil {
		payload.StatusCode = utils.ErrRefuse
		payload.Message = "服务器爆满，请稍后再试"
		return err
	}
	if hub.ExplicitStop {
		payload.StatusCode = utils.ErrRefuse
		payload.Message = "服务器繁忙，请稍后再试"
		return err
	}

	conn, err := upgrade.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		tmpStr := fmt.Sprintf("Handshake |upgrade err: %v", err)
		glog.Warnf(tmpStr)
		return errors.New(tmpStr)
	}

	defer func() {
		if err != nil {
			glog.Emergency("Handshake error|err=%v", err)

			err = conn.WriteJSON(sendMessage)
			if err != nil {
				glog.Errorf("Handshake |WriteMessage err: %v |data=%v", err, sendMessage)
			}
			err1 := conn.Close()
			if err1 != nil {
				glog.Errorf("Handshake |wsClose err: %v", err)
			}
		}
	}()

	client := &Client{
		Id:             randid.GenerateId(),
		hub:            hub,
		clientType:     clientType,
		conn:           conn,
		siteId:         siteId,
		send:           make(chan []byte, 2000),
		ip:             libip.ClientIP(c),
		state:          statePending,
		server:         utils.GetLocalIP(),
		userAgent:      c.Request.Header.Get("User-Agent"),
		creatAt:        time.Now().Unix(),
		property:       make(map[string]interface{}),
		RoomPushMethod: pushToKafka,
	}
	client.ctx, client.cancel = ctx.WithCancel(ctx.Background())
	hub.Add(client)
	go client.writePumpWss()
	go client.readPumpWss()
	app.onConnectionCreate(client)
	return err
}
