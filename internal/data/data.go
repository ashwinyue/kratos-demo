package data

import (
	"context"
	"time"

	"kratos-demo/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGreeterRepo)

// Data .
type Data struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
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
		db:  db,
		rdb: rdb,
	}
	
	cleanup := func() {
		helper.Info("closing the data resources")
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		rdb.Close()
	}
	
	return data, cleanup, nil
}
