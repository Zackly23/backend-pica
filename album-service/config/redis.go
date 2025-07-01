package config

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client
var Ctx = context.Background()

func ConnectRedis() {
	Redis = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"), // contoh: localhost:6379
		Password: os.Getenv("REDIS_PASSWORD"), // kosongin kalau nggak pakai password
		DB:       0,
	})

	_, err := Redis.Ping(Ctx).Result()
	if err != nil {
		log.Fatalf("Unable to connect to Redis: %v\n", err)
	}

	fmt.Println("✅ Redis connected")
}
