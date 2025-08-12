package service

import (
	"context"

	"kratos-demo/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// MessageService 消息服务
type MessageService struct {
	messageUc *biz.MessageUsecase
	logger    log.Logger
}

// NewMessageService 创建消息服务
func NewMessageService(messageUc *biz.MessageUsecase, logger log.Logger) *MessageService {
	return &MessageService{
		messageUc: messageUc,
		logger:    logger,
	}
}

// SendMessage 发送消息
func (s *MessageService) SendMessage(ctx context.Context, topic, tag string, data interface{}) error {
	return s.messageUc.SendMessage(ctx, topic, tag, data)
}

// SendAsyncMessage 异步发送消息
func (s *MessageService) SendAsyncMessage(ctx context.Context, topic, tag string, data interface{}, callback func(ctx context.Context, result interface{}, err error)) error {
	return s.messageUc.SendAsyncMessage(ctx, topic, tag, data, callback)
}

// SubscribeMessage 订阅消息
func (s *MessageService) SubscribeMessage(topic, selector string, handler func(context.Context, ...interface{}) error) error {
	return s.messageUc.SubscribeMessage(topic, selector, handler)
}

// HandleGreeterMessage 处理greeter消息
func (s *MessageService) HandleGreeterMessage(ctx context.Context, data []byte) error {
	return s.messageUc.ProcessGreeterMessage(ctx, data)
}