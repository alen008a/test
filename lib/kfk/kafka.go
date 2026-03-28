package kfk

import (
	"msgPushSite/internal/glog"
)

var (
	msgPushTopic = "msg_push_site_kfk_topic"
	addrArr      []string
)

// InitKafka 初始化 kafka
func InitKafka() error {
	// 1. 初始化生产者
	err := initProducer()
	if err != nil {
		glog.Emergency("kafka initProducer error|err=>%v", err)
		return err
	}

	// 2. 初始化消费者组
	err = initConsumerGroup()
	if err != nil {
		glog.Emergency("kafka initConsumerGroup error|err=>%v", err)
		return err
	}
	return nil
}

// Close 关闭 kafka的接口
func Close() {
	if asyncProducer != nil {
		asyncProducer.Close()
	}
	if cGroup != nil {
		cGroup.Close()
	}
	glog.Info("Kafka已安全退出~~")
}
