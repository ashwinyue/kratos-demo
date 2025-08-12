package service

import (
	"context"

	"kratos-demo/internal/message"

	"github.com/go-kratos/kratos/v2/log"
)

// MessageService 消息服务
type MessageService struct {
	producer message.Producer
	consumer message.Consumer
	handler  message.MessageHandler
	logger   log.Logger
}

// NewMessageService 创建消息服务
func NewMessageService(producer message.Producer, consumer message.Consumer, handler message.MessageHandler, logger log.Logger) *MessageService {
	return &MessageService{
		producer: producer,
		consumer: consumer,
		handler:  handler,
		logger:   logger,
	}
}

// SendMessage 发送消息
func (s *MessageService) SendMessage(ctx context.Context, topic, tag string, data interface{}) error {
	return s.producer.SendMessage(ctx, topic, tag, data)
}

// SendAsyncMessage 异步发送消息
func (s *MessageService) SendAsyncMessage(ctx context.Context, topic, tag string, data interface{}, callback func(ctx context.Context, result interface{}, err error)) error {
	return s.producer.SendAsyncMessage(ctx, topic, tag, data, callback)
}

// SendOneWayMessage 单向发送消息
func (s *MessageService) SendOneWayMessage(ctx context.Context, topic, tag string, data interface{}) error {
	return s.producer.SendOneWayMessage(ctx, topic, tag, data)
}

// SubscribeMessage 订阅消息
func (s *MessageService) SubscribeMessage(topic, selector string, handler func(context.Context, ...interface{}) error) error {
	return s.consumer.SubscribeMessage(topic, selector, handler)
}

// StartConsumer 启动消费者
func (s *MessageService) StartConsumer() error {
	return s.consumer.StartConsumer()
}

// StopConsumer 停止消费者
func (s *MessageService) StopConsumer() error {
	return s.consumer.StopConsumer()
}

// HandleGreeterMessage 处理greeter消息
func (s *MessageService) HandleGreeterMessage(ctx context.Context, data []byte) error {
	return s.handler.ProcessMessage(ctx, "greeter", "", data)
}