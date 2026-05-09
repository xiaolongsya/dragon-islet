package initialize

import (
	"context"
	"dragon-islet/internal/global"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

func InitRedis() {
	addr := global.CONFIG.GetString("redis.addr")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	password := global.CONFIG.GetString("redis.password")

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}

	global.REDIS = client
	fmt.Println("Redis initialized successfully.")
}
