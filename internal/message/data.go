package message

import (
	"context"
	"encoding/json"
	"kratos-demo/internal/conf"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/go-kratos/kratos/v2/log"
)

// messageProducer 消息生产者实现
type messageProducer struct {
	producer rocketmq.Producer
	logger   log.Logger
}

// messageConsumer 消息消费者实现
type messageConsumer struct {
	consumer rocketmq.PushConsumer
	logger   log.Logger
}

// NewMessageProducer 创建消息生产者
func NewMessageProducer(c *conf.Data, logger log.Logger) (Producer, error) {
	helper := log.NewHelper(logger)

	options := []producer.Option{
		producer.WithNameServer(c.Rocketmq.NameServers),
		producer.WithGroupName(c.Rocketmq.GroupName),
	}

	if c.Rocketmq.Namespace != "" {
		options = append(options, producer.WithNamespace(c.Rocketmq.Namespace))
	}

	if c.Rocketmq.AccessKey != "" && c.Rocketmq.SecretKey != "" {
		options = append(options, producer.WithCredentials(primitive.Credentials{
			AccessKey: c.Rocketmq.AccessKey,
			SecretKey: c.Rocketmq.SecretKey,
		}))
	}

	prod, err := rocketmq.NewProducer(options...)
	if err != nil {
		helper.Errorf("failed to create rocketmq producer: %v", err)
		return nil, err
	}

	if err := prod.Start(); err != nil {
		helper.Errorf("failed to start rocketmq producer: %v", err)
		return nil, err
	}

	helper.Info("rocketmq producer started successfully")
	return &messageProducer{
		producer: prod,
		logger:   logger,
	}, nil
}

// NewMessageConsumer 创建消息消费者
func NewMessageConsumer(c *conf.Data, logger log.Logger) (Consumer, error) {
	helper := log.NewHelper(logger)

	options := []consumer.Option{
		consumer.WithNameServer(c.Rocketmq.NameServers),
		consumer.WithGroupName(c.Rocketmq.GroupName),
	}

	if c.Rocketmq.Namespace != "" {
		options = append(options, consumer.WithNamespace(c.Rocketmq.Namespace))
	}

	if c.Rocketmq.AccessKey != "" && c.Rocketmq.SecretKey != "" {
		options = append(options, consumer.WithCredentials(primitive.Credentials{
			AccessKey: c.Rocketmq.AccessKey,
			SecretKey: c.Rocketmq.SecretKey,
		}))
	}

	cons, err := rocketmq.NewPushConsumer(options...)
	if err != nil {
		helper.Errorf("failed to create rocketmq consumer: %v", err)
		return nil, err
	}

	helper.Info("rocketmq consumer created successfully")
	return &messageConsumer{
		consumer: cons,
		logger:   logger,
	}, nil
}

// SendMessage 发送消息
func (m *messageProducer) SendMessage(ctx context.Context, topic, tag string, data interface{}) error {
	helper := log.NewHelper(m.logger)

	if m.producer == nil {
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

	result, err := m.producer.SendSync(ctx, msg)
	if err != nil {
		helper.Errorf("failed to send message: %v", err)
		return err
	}

	helper.Infof("message sent successfully, msgId: %s, topic: %s, tag: %s",
		result.MsgID, topic, tag)
	return nil
}

// SendAsyncMessage 异步发送消息
func (m *messageProducer) SendAsyncMessage(ctx context.Context, topic, tag string, data interface{}, callback func(ctx context.Context, result interface{}, err error)) error {
	helper := log.NewHelper(m.logger)

	if m.producer == nil {
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

	err = m.producer.SendAsync(ctx, wrappedCallback, msg)
	if err != nil {
		helper.Errorf("failed to send async message: %v", err)
		return err
	}

	helper.Infof("async message sent, topic: %s, tag: %s", topic, tag)
	return nil
}

// SendOneWayMessage 单向发送消息（不关心结果）
func (m *messageProducer) SendOneWayMessage(ctx context.Context, topic, tag string, data interface{}) error {
	helper := log.NewHelper(m.logger)

	if m.producer == nil {
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

	err = m.producer.SendOneWay(ctx, msg)
	if err != nil {
		helper.Errorf("failed to send oneway message: %v", err)
		return err
	}

	helper.Infof("oneway message sent, topic: %s, tag: %s", topic, tag)
	return nil
}

// SubscribeMessage 订阅消息
func (m *messageConsumer) SubscribeMessage(topic, selector string, handler func(context.Context, ...interface{}) error) error {
	helper := log.NewHelper(m.logger)

	if m.consumer == nil {
		helper.Error("rocketmq consumer is not available")
		return nil
	}

	err := m.consumer.Subscribe(topic, consumer.MessageSelector{
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
func (m *messageConsumer) StartConsumer() error {
	helper := log.NewHelper(m.logger)

	if m.consumer == nil {
		helper.Info("rocketmq consumer is not available")
		return nil
	}

	err := m.consumer.Start()
	if err != nil {
		helper.Errorf("failed to start consumer: %v", err)
		return err
	}

	helper.Info("rocketmq consumer started successfully")
	return nil
}

// StopConsumer 停止消费者
func (m *messageConsumer) StopConsumer() error {
	helper := log.NewHelper(m.logger)

	if m.consumer == nil {
		helper.Info("rocketmq consumer is not available")
		return nil
	}

	err := m.consumer.Shutdown()
	if err != nil {
		helper.Errorf("failed to stop consumer: %v", err)
		return err
	}

	helper.Info("rocketmq consumer stopped successfully")
	return nil
}

