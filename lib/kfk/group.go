package kfk

import (
	ctx "context"
	"github.com/RussellLuo/timingwheel"
	"github.com/Shopify/sarama"
	"msgPushSite/config"
	"msgPushSite/internal/glog"
	"msgPushSite/lib/ws"
	"msgPushSite/mdata"
	"os"
	"strings"
	"time"
)

var cGroup sarama.ConsumerGroup

func initConsumerGroup() error {
	var err error
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Errors = true // 是否等待成功和失败后的响应,只有上面的 RequiredAcks 设置不是 NoReponse 这里才有用
	saramaConfig.Version = sarama.V2_6_0_0     //设置使用的kafka版本,如果低于V0_10_0_0版本,消息中的timestrap没有作用.需要消费和生产同时配置
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	kafkaConfig := config.GetKafkaConfig()
	//这里每个节点都必须消费全量kafka数据，必须要使用不同的groupID
	//原因是消费完之后要通过ws推送给客户端，而每个ws服务端只连接了一部分客户端。
	groupId, err := os.Hostname()
	if err != nil {
		glog.Fatalf("Get host name is error: %s", err.Error())
	}
	cGroup, err = sarama.NewConsumerGroup(strings.Split(kafkaConfig.KafkaAddr, ","), groupId, saramaConfig)
	if err != nil {
		return err
	}
	groupHandler := &GroupHandler{ready: make(chan bool)}
	go func() {
		for {
			err = cGroup.Consume(ctx.Background(), strings.Split(config.GetKafkaTopic().ChatMsgWriteTopic, ","), groupHandler)
			if err != nil {
				glog.Error("Error from consumer: %v", err)
			}
			groupHandler.ready = make(chan bool)
		}
	}()
	glog.Info("Init kafka consumer group success!!!")
	return nil
}

type GroupHandler struct {
	ready chan bool
}

func (g *GroupHandler) Setup(sarama.ConsumerGroupSession) error {
	glog.Infof("kfk consumer setup")
	close(g.ready)
	return nil
}

func (g *GroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	glog.Infof("kfk consumer cleanup")
	return nil
}

func (g *GroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		glog.Infof("Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
		session.MarkMessage(message, "")
		var packet = ws.PayloadBytes(message.Value)

		var (
			timeoutChan = make(chan struct{})
			tw          *timingwheel.Timer
		)

		tw = mdata.TimingWheel.AfterFunc(time.Second*5, func() { timeoutChan <- struct{}{} })

		select {
		case ws.MsgChan <- packet:
		case <-timeoutChan: // 增加一个超时机制
			glog.Error("------- push kafka message to MsgChan timeout")
		}
		tw.Stop()
	}
	return nil
}
