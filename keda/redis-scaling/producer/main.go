package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

const (
	redisListKey = "workqueue"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()

	// Get Redis connection details from environment
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		logger.Fatal("REDIS_HOST environment variable not set")
	}

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: redisHost,
		DB:   0,
	})
	defer rdb.Close()

	ctx := context.Background()

	// Test connection
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}

	// Get message count from environment
	count, err := strconv.Atoi(os.Getenv("COUNT"))
	if err != nil {
		logger.Error("invalid count provided, using default", zap.Error(err))
		count = 1 // Default to 1 messages
	}

	// Get batch size from environment
	batchSize, err := strconv.Atoi(os.Getenv("BATCH_SIZE"))
	if err != nil {
		logger.Error("invalid batch size provided, using default", zap.Error(err))
		batchSize = 1 // Default to 1 messages per batch
	}

	// Continuously produce messages in batches
	messageID := 1
	for {
		// Create a batch of messages
		messages := make([]interface{}, batchSize)
		for i := 0; i < batchSize; i++ {
			messages[i] = "Test" + strconv.Itoa(messageID)
			messageID++

			// Reset counter if we've reached count
			if messageID > count {
				messageID = 1
			}
		}

		// Push batch of messages to Redis
		err := rdb.RPush(ctx, redisListKey, messages...).Err()
		if err != nil {
			logger.Error("failed to push messages",
				zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		// Log batch completion
		logger.Info("batch of messages pushed",
			zap.Int("batch_size", batchSize),
			zap.Int("last_message_id", messageID-1))

		// Get current list length
		length, err := rdb.LLen(ctx, redisListKey).Result()
		if err != nil {
			logger.Error("failed to get list length", zap.Error(err))
		} else {
			logger.Info("current queue length", zap.Int64("length", length))
		}

		// Small delay between batches
		time.Sleep(500 * time.Millisecond)
	}
}
