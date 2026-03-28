package kfk

import (
	"errors"
	"fmt"
	"time"

	"msgPushSite/internal/glog"
	"msgPushSite/lib/ws"
	"msgPushSite/mdata"

	"github.com/RussellLuo/timingwheel"

	"github.com/Shopify/sarama"
)

// initConsumer 初始化消费者
func initConsumer() error {
	config := sarama.NewConfig()
	config.Producer.Return.Errors = true // 是否等待成功和失败后的响应,只有上面的 RequiredAcks 设置不是 NoReponse 这里才有用
	config.Version = sarama.V2_6_0_0     //设置使用的kafka版本,如果低于V0_10_0_0版本,消息中的timestrap没有作用.需要消费和生产同时配置
	addrArr = []string{"18.163.178.90:9092"}
	client, err := sarama.NewClient(addrArr, config)
	if err != nil {
		tmpStr := fmt.Sprintf("create kafkaClient addr=%v err: %v", addrArr, err)
		return errors.New(tmpStr)
	}

	// 消息推送topic 的所有分区
	partArr, err := client.Partitions(msgPushTopic)
	if err != nil {
		tmpStr := fmt.Sprintf("get topc=%s partitions err: %v", msgPushTopic, err)
		return errors.New(tmpStr)
	}

	for i := range partArr {
		part := partArr[i]
		go initIMKafkaConsumer(msgPushTopic, part, sarama.OffsetNewest)
	}

	return nil
}

// initKafkaConsumer kafka 消费者
func initIMKafkaConsumer(topic string, partition int32, offset int64) {
	var err error

	config := sarama.NewConfig()
	config.Producer.Return.Errors = true // 是否等待成功和失败后的响应,只有上面的 RequiredAcks 设置不是 NoReponse 这里才有用
	config.Version = sarama.V2_6_0_0     //设置使用的kafka版本,如果低于V0_10_0_0版本,消息中的timestrap没有作用.需要消费和生产同时配置

	// 新建一个消费者
	imConsumer, err := sarama.NewConsumer(addrArr, config)
	if err != nil {
		glog.Emergency("create consumer err: %v", err)
		return
	}

	// 根据消费者获取指定的主题分区的消费者, offset 为偏移量, sarama.OffsetOldest: 为从头开始消费, sarama.OffsetNewest 为从最新的偏移量开始消费, 0: 即获当前已消费的偏移量
	partitionConsumer, err := imConsumer.ConsumePartition(topic, partition, offset)
	if err != nil {
		tmpStr := fmt.Sprintf("get partition consumer err: %v", err)
		glog.Errorf(tmpStr)
		return
	}
	defer partitionConsumer.Close()

	// 循环等待接收信息
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			msgStr := string(msg.Value)
			//tmpStr := fmt.Sprintf("msg offset: %v partition: %v value: %v", msg.Offset, msg.Partition, msgStr)
			glog.Infof("kafkaConsumer: %s", msgStr)

			var packet = ws.PayloadBytes(msg.Value)

			var (
				timeoutChan = make(chan struct{})
				tw          *timingwheel.Timer
			)

			tw = mdata.TimingWheel.AfterFunc(time.Second*5, func() { timeoutChan <- struct{}{} })

			select {
			case ws.MsgChan <- packet:
			case <-timeoutChan: // 增加一个超时机制
			}
			tw.Stop()
		case err := <-partitionConsumer.Errors():
			glog.Errorf("accept partition err: %v", err)
		}
	}
}
