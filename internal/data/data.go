package data

import (
	"context"
	"time"

	"kratos-demo/internal/conf"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGreeterRepo, NewMongoDB, NewMongoGreeterRepo)

// Data .
type Data struct {
	db       *gorm.DB
	rdb      *redis.Client
	mongodb  *mongo.Database
}

// NewData .
func NewData(c *conf.Data, logger log.Logger, mongodb *mongo.Database) (*Data, func(), error) {
	helper := log.NewHelper(logger)

	// Initialize MySQL
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		helper.Fatalf("failed to connect to database: %v", err)
		return nil, nil, err
	}

	// Initialize Redis
	rdb := redis.NewClient(&redis.Options{
		Network:      c.Redis.Network,
		Addr:         c.Redis.Addr,
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		helper.Errorf("failed to connect to redis: %v", err)
	}

	data := &Data{
		db:      db,
		rdb:     rdb,
		mongodb: mongodb,
	}

	cleanup := func() {
		helper.Info("closing the data resources")
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		rdb.Close()
		if mongodb != nil {
			mongodb.Client().Disconnect(context.Background())
		}
	}

	return data, cleanup, nil
}

// NewMongoDB creates a new MongoDB database connection
func NewMongoDB(c *conf.Data, logger log.Logger) (*mongo.Database, error) {
	helper := log.NewHelper(logger)

	ctx, cancel := context.WithTimeout(context.Background(), c.Mongodb.Timeout.AsDuration())
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(c.Mongodb.Uri))
	if err != nil {
		helper.Errorf("failed to connect to mongodb: %v", err)
		return nil, err
	}

	// Test connection
	if err := client.Ping(ctx, nil); err != nil {
		helper.Errorf("failed to ping mongodb: %v", err)
		return nil, err
	}

	helper.Info("connected to mongodb successfully")
	return client.Database(c.Mongodb.Database), nil
}

// NewRocketMQProducer creates a new RocketMQ producer
func NewRocketMQProducer(c *conf.Data, logger log.Logger) (rocketmq.Producer, error) {
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
	return prod, nil
}

// NewRocketMQConsumer creates a new RocketMQ consumer
func NewRocketMQConsumer(c *conf.Data, logger log.Logger) (rocketmq.PushConsumer, error) {
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
	return cons, nil
}
