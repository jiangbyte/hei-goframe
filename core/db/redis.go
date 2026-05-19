package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"hei-gin/config"
)

var Redis *redis.Client

func InitRedis() error {
	cfg := config.C.Redis
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	Redis = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.Database,
		PoolSize:     cfg.MaxConnections,
		DialTimeout:  time.Duration(cfg.SocketConnectTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.SocketTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.SocketTimeout) * time.Second,
	})

	ctx := context.Background()
	if err := Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	log.Println("[Database] Redis connection verified")
	return nil
}

func CloseRedis() {
	if Redis != nil {
		Redis.Close()
		Redis = nil
	}
}
