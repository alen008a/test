package kfk

import (
	"fmt"
	"strings"

	"msgPushSite/internal/glog"

	"msgPushSite/config"

	"github.com/Shopify/sarama"
)

var asyncProducer sarama.AsyncProducer

// 初始化 kafka 的生产者
func initProducer() error {
	saConfig := sarama.NewConfig()
	saConfig.Producer.RequiredAcks = sarama.WaitForAll          // 等待服务器所有副本都保存成功后才响应
	saConfig.Producer.Partitioner = sarama.NewRandomPartitioner // 随机的分区类型
	saConfig.Producer.Return.Successes = true                   // 是否等待成功和失败后的响应,只有上面的 RequiredAcks 设置不是 NoReponse 这里才有用
	saConfig.Producer.Return.Errors = true                      // 开启异常捕获
	saConfig.Version = sarama.V2_6_0_0                          // 设置使用的kafka版本,如果低于V0_10_0_0版本,消息中的timestrap没有作用.需要消费和生产同时配置
	// 异步生产者
	go AsyncProducer(saConfig)

	return nil
}

// AsyncProducer 异步 kafka 的生产者
func AsyncProducer(saConfig *sarama.Config) error {
	fmt.Println("kafka brokers is :", config.GetKafkaConfig().KafkaAddr)
	addrArr = strings.Split(config.GetKafkaConfig().KafkaAddr, ",")
	var err error

	// 使用配置，新建一个异步生产者
	asyncProducer, err = sarama.NewAsyncProducer(addrArr, saConfig)
	if err != nil {
		return err
	}

	for {
		// 判断发送是否成功
		select {
		case <-asyncProducer.Successes():
		case err = <-asyncProducer.Errors():
			glog.Errorf("Kafka push message is error: %s", err.Error())
		}
	}
}

// MsgPushKafka 消息推送kafka中
func MsgPushKafka(data []byte, topic string, key ...string) {
	var k string
	if len(key) != 1 {
		k = topic
	} else {
		k = key[0]
	}

	if asyncProducer == nil {
		return
	}
	//  发送的消息， 主题， key
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(k),
		Value: sarama.StringEncoder(data),
	}
	asyncProducer.Input() <- msg
}
