package message

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is message providers.
var ProviderSet = wire.NewSet(NewDefaultMessageHandler)

// Message 消息实体
type Message struct {
	Topic string
	Tag   string
	Body  []byte
	ID    string
}

// Producer 消息生产者接口
type Producer interface {
	// SendMessage 发送消息
	SendMessage(ctx context.Context, topic, tag string, data interface{}) error
	// SendAsyncMessage 异步发送消息
	SendAsyncMessage(ctx context.Context, topic, tag string, data interface{}, callback func(ctx context.Context, result interface{}, err error)) error
	// SendOneWayMessage 单向发送消息（不关心结果）
	SendOneWayMessage(ctx context.Context, topic, tag string, data interface{}) error
}

// Consumer 消息消费者接口
type Consumer interface {
	// SubscribeMessage 订阅消息
	SubscribeMessage(topic, selector string, handler func(context.Context, ...interface{}) error) error
	// StartConsumer 启动消费者
	StartConsumer() error
	// StopConsumer 停止消费者
	StopConsumer() error
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	// ProcessMessage 处理消息的业务逻辑
	ProcessMessage(ctx context.Context, topic, tag string, data []byte) error
}

// DefaultMessageHandler 默认消息处理器
type DefaultMessageHandler struct {
	logger log.Logger
}

// NewDefaultMessageHandler 创建默认消息处理器
func NewDefaultMessageHandler(logger log.Logger) *DefaultMessageHandler {
	return &DefaultMessageHandler{
		logger: logger,
	}
}

// ProcessMessage 处理消息的默认实现
func (h *DefaultMessageHandler) ProcessMessage(ctx context.Context, topic, tag string, data []byte) error {
	helper := log.NewHelper(h.logger)
	helper.Infof("processing message from topic: %s, tag: %s, data: %s", topic, tag, string(data))
	
	// 这里可以根据topic和tag进行不同的业务处理
	switch topic {
	case "greeter":
		return h.processGreeterMessage(ctx, tag, data)
	default:
		helper.Warnf("unknown topic: %s", topic)
	}
	
	return nil
}

// processGreeterMessage 处理greeter消息
func (h *DefaultMessageHandler) processGreeterMessage(ctx context.Context, tag string, data []byte) error {
	helper := log.NewHelper(h.logger)
	helper.Infof("processing greeter message with tag: %s, data: %s", tag, string(data))
	
	// 具体的业务逻辑处理
	// 比如解析消息、调用其他服务、更新数据库等
	
	return nil
}