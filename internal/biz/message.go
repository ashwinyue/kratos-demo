package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// Message 消息实体
type Message struct {
	Topic string
	Tag   string
	Body  []byte
	ID    string
}

// MessageRepo 消息仓储接口
type MessageRepo interface {
	// SendMessage 发送消息
	SendMessage(ctx context.Context, topic, tag string, data interface{}) error
	// SendAsyncMessage 异步发送消息
	SendAsyncMessage(ctx context.Context, topic, tag string, data interface{}, callback func(ctx context.Context, result interface{}, err error)) error
	// SubscribeMessage 订阅消息
	SubscribeMessage(topic, selector string, handler func(context.Context, ...interface{}) error) error
}

// MessageUsecase 消息用例
type MessageUsecase struct {
	repo   MessageRepo
	logger log.Logger
}

// NewMessageUsecase 创建消息用例
func NewMessageUsecase(repo MessageRepo, logger log.Logger) *MessageUsecase {
	return &MessageUsecase{
		repo:   repo,
		logger: logger,
	}
}

// SendMessage 发送消息
func (uc *MessageUsecase) SendMessage(ctx context.Context, topic, tag string, data interface{}) error {
	helper := log.NewHelper(uc.logger)
	helper.Infof("sending message to topic: %s, tag: %s", topic, tag)
	
	return uc.repo.SendMessage(ctx, topic, tag, data)
}

// SendAsyncMessage 异步发送消息
func (uc *MessageUsecase) SendAsyncMessage(ctx context.Context, topic, tag string, data interface{}, callback func(ctx context.Context, result interface{}, err error)) error {
	helper := log.NewHelper(uc.logger)
	helper.Infof("sending async message to topic: %s, tag: %s", topic, tag)
	
	return uc.repo.SendAsyncMessage(ctx, topic, tag, data, callback)
}

// SubscribeMessage 订阅消息
func (uc *MessageUsecase) SubscribeMessage(topic, selector string, handler func(context.Context, ...interface{}) error) error {
	helper := log.NewHelper(uc.logger)
	helper.Infof("subscribing to topic: %s, selector: %s", topic, selector)
	
	return uc.repo.SubscribeMessage(topic, selector, handler)
}

// ProcessGreeterMessage 处理greeter消息的业务逻辑
func (uc *MessageUsecase) ProcessGreeterMessage(ctx context.Context, data []byte) error {
	helper := log.NewHelper(uc.logger)
	helper.Infof("processing greeter message: %s", string(data))
	
	// 这里可以添加具体的业务逻辑
	// 比如解析消息、调用其他服务、更新数据库等
	
	return nil
}