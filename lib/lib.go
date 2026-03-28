package lib

import (
	"fmt"
	"msgPushSite/internal/glog"
	libip "msgPushSite/lib/ip"
	"msgPushSite/lib/kfk"
	"msgPushSite/lib/ws"
)

func InitLib() error {
	err := libip.InitIP()
	if err != nil {
		glog.Error(err)
		return err
	}

	err = kfk.InitKafka()
	if err != nil {
		return err
	}
	ws.InitApp(kfk.MsgPushKafka)
	fmt.Println("init ws app success!!!!!")
	return nil
}

func Close() {
	// 1. 退出长连接管理
	ws.AllHubStop()
	// 2. 退出Kafka-Produce
	kfk.Close()
}
