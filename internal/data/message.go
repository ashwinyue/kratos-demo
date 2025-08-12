package data

import (
	"context"
	"encoding/json"

	"kratos-demo/internal/biz"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/go-kratos/kratos/v2/log"
)

// messageRepo 消息仓储实现
type messageRepo struct {
	data   *Data
	logger log.Logger
}

// NewMessageRepo 创建消息仓储
func NewMessageRepo(data *Data, logger log.Logger) biz.MessageRepo {
	return &messageRepo{
		data:   data,
		logger: logger,
	}
}

// SendMessage 发送消息
func (m *messageRepo) SendMessage(ctx context.Context, topic, tag string, data interface{}) error {
	helper := log.NewHelper(m.logger)

	if m.data.mqProd == nil {
		helper.Error("rocketmq producer is not available")
		return nil
	}

	body, err := json.Marshal(data)
	if err != nil {
		helper.Errorf("failed to marshal message data: %v", err)
		return err
	}

	msg := &primitive.Message{
		Topic: topic,
		Body:  body,
	}
	msg.WithTag(tag)

	result, err := m.data.mqProd.SendSync(ctx, msg)
	if err != nil {
		helper.Errorf("failed to send message: %v", err)
		return err
	}

	helper.Infof("message sent successfully, msgId: %s, topic: %s, tag: %s", 
		result.MsgID, topic, tag)
	return nil
}

// SendAsyncMessage 异步发送消息
func (m *messageRepo) SendAsyncMessage(ctx context.Context, topic, tag string, data interface{}, callback func(ctx context.Context, result interface{}, err error)) error {
	helper := log.NewHelper(m.logger)

	if m.data.mqProd == nil {
		helper.Error("rocketmq producer is not available")
		return nil
	}

	body, err := json.Marshal(data)
	if err != nil {
		helper.Errorf("failed to marshal message data: %v", err)
		return err
	}

	msg := &primitive.Message{
		Topic: topic,
		Body:  body,
	}
	msg.WithTag(tag)

	// 包装回调函数以适配接口
	wrappedCallback := func(ctx context.Context, result *primitive.SendResult, err error) {
		callback(ctx, result, err)
	}

	err = m.data.mqProd.SendAsync(ctx, wrappedCallback, msg)
	if err != nil {
		helper.Errorf("failed to send async message: %v", err)
		return err
	}

	helper.Infof("async message sent, topic: %s, tag: %s", topic, tag)
	return nil
}

// SendOneWayMessage 单向发送消息（不关心结果）
func (m *messageRepo) SendOneWayMessage(ctx context.Context, topic, tag string, data interface{}) error {
	helper := log.NewHelper(m.logger)

	if m.data.mqProd == nil {
		helper.Error("rocketmq producer is not available")
		return nil
	}

	body, err := json.Marshal(data)
	if err != nil {
		helper.Errorf("failed to marshal message data: %v", err)
		return err
	}

	msg := &primitive.Message{
		Topic: topic,
		Body:  body,
	}
	msg.WithTag(tag)

	err = m.data.mqProd.SendOneWay(ctx, msg)
	if err != nil {
		helper.Errorf("failed to send oneway message: %v", err)
		return err
	}

	helper.Infof("oneway message sent, topic: %s, tag: %s", topic, tag)
	return nil
}

// SubscribeMessage 订阅消息
func (m *messageRepo) SubscribeMessage(topic, selector string, handler func(context.Context, ...interface{}) error) error {
	helper := log.NewHelper(m.logger)

	if m.data.mqCons == nil {
		helper.Error("rocketmq consumer is not available")
		return nil
	}

	err := m.data.mqCons.Subscribe(topic, consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: selector,
	}, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		helper.Infof("received %d messages from topic: %s", len(msgs), topic)
		
		// 转换消息类型以适配接口
		interfaceMsgs := make([]interface{}, len(msgs))
		for i, msg := range msgs {
			interfaceMsgs[i] = msg
		}
		
		if err := handler(ctx, interfaceMsgs...); err != nil {
			helper.Errorf("failed to handle messages: %v", err)
			return consumer.ConsumeRetryLater, err
		}
		
		return consumer.ConsumeSuccess, nil
	})

	if err != nil {
		helper.Errorf("failed to subscribe topic %s: %v", topic, err)
		return err
	}

	helper.Infof("subscribed to topic: %s, selector: %s", topic, selector)
	return nil
}

// StartConsumer 启动消费者
func (m *messageRepo) StartConsumer() error {
	helper := log.NewHelper(m.logger)

	if m.data.mqCons == nil {
		helper.Info("rocketmq consumer is not available")
		return nil
	}

	err := m.data.mqCons.Start()
	if err != nil {
		helper.Errorf("failed to start consumer: %v", err)
		return err
	}

	helper.Info("rocketmq consumer started successfully")
	return nil
}

// StopConsumer 停止消费者
func (m *messageRepo) StopConsumer() error {
	helper := log.NewHelper(m.logger)

	if m.data.mqCons == nil {
		helper.Info("rocketmq consumer is not available")
		return nil
	}

	err := m.data.mqCons.Shutdown()
	if err != nil {
		helper.Errorf("failed to stop consumer: %v", err)
		return err
	}

	helper.Info("rocketmq consumer stopped successfully")
	return nil
}