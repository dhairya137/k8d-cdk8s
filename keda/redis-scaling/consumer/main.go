package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

const (
	redisListKey = "workqueue"
	batchSize    = 50
)

// Track processed messages to avoid duplicates
var (
	processedMessages = make(map[string]bool)
	processedMutex   sync.RWMutex
)

func isProcessed(msg string) bool {
	processedMutex.RLock()
	defer processedMutex.RUnlock()
	return processedMessages[msg]
}

func markProcessed(msg string) {
	processedMutex.Lock()
	defer processedMutex.Unlock()
	processedMessages[msg] = true
}

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test connection
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}

	// Get max retries from environment
	maxRetries, err := strconv.Atoi(os.Getenv("MAX_RETRIES"))
	if err != nil {
		logger.Error("invalid max retries provided, using default", zap.Error(err))
		maxRetries = 3
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Process messages until shutdown signal
	for {
		select {
		case <-sigChan:
			logger.Info("received shutdown signal")
			return
		default:
			// Get current list length
			length, err := rdb.LLen(ctx, redisListKey).Result()
			if err != nil {
				logger.Error("failed to get list length", zap.Error(err))
				time.Sleep(time.Second)
				continue
			}

			if length == 0 {
				time.Sleep(100 * time.Millisecond) // Reduced sleep time when no messages
				continue
			}

			// Try to get messages in batch using LRANGE
			messages, err := rdb.LRange(ctx, redisListKey, 0, batchSize-1).Result()
			if err != nil {
				logger.Error("failed to get messages", zap.Error(err))
				time.Sleep(100 * time.Millisecond)
				continue
			}

			processedCount := 0
			successfulMessages := make([]string, 0, len(messages))

			// Process each message in the batch
			for _, msg := range messages {
				// Skip if already processed
				if isProcessed(msg) {
					continue
				}

				// Process message with retries
				success := false
				for attempt := 0; attempt < maxRetries; attempt++ {
					// Log processing attempt
					logger.Info("processing message",
						zap.String("message", msg),
						zap.Int("attempt", attempt+1))

					// Simulate processing time (reduced)
					time.Sleep(10 * time.Millisecond)

					// Simulate random processing failures (10% chance)
					if attempt < maxRetries-1 && time.Now().UnixNano()%10 == 0 {
						logger.Error("processing failed, retrying",
							zap.String("message", msg),
							zap.Int("attempt", attempt+1))
						continue
					}

					// Mark as processed
					markProcessed(msg)
					processedCount++
					successfulMessages = append(successfulMessages, msg)
					success = true
					break
				}

				if !success {
					logger.Error("message processing failed after max retries",
						zap.String("message", msg),
						zap.Int("max_retries", maxRetries))
				}
			}

			// Remove all successfully processed messages in one operation
			if len(successfulMessages) > 0 {
				pipe := rdb.Pipeline()
				for _, msg := range successfulMessages {
					pipe.LRem(ctx, redisListKey, 1, msg)
				}
				_, err := pipe.Exec(ctx)
				if err != nil {
					logger.Error("failed to remove processed messages",
						zap.Error(err))
				}
			}

			// Log batch processing summary
			if processedCount > 0 {
				logger.Info("batch processing completed",
					zap.Int("processed_count", processedCount),
					zap.Int64("remaining_messages", length-int64(processedCount)))
			}

			// Cleanup old processed messages (keep last 1000)
			if len(processedMessages) > 1000 {
				processedMutex.Lock()
				processedMessages = make(map[string]bool)
				processedMutex.Unlock()
			}
		}
	}
}
