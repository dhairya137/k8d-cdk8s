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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.uber.org/zap"
)

const (
	batchSize = 10 // Maximum SQS batch size is 10
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()

	// Get queue URL from environment
	queueURL := os.Getenv("QUEUE_URL")
	if queueURL == "" {
		logger.Fatal("QUEUE_URL environment variable not set")
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Fatal("unable to load SDK config", zap.Error(err))
	}

	// Create SQS client
	sqsClient := sqs.NewFromConfig(cfg)

	// Get max retries from environment
	maxRetries, err := strconv.Atoi(os.Getenv("MAX_RETRIES"))
	if err != nil {
		logger.Error("invalid max retries provided, using default", zap.Error(err))
		maxRetries = 3
	}

	// Setup context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start processing messages
	var wg sync.WaitGroup
	processMessages := make(chan types.Message, batchSize)

	// Start worker goroutines for processing messages
	numWorkers := 3
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for msg := range processMessages {
				processMessage(ctx, logger, sqsClient, queueURL, msg, maxRetries, workerID)
			}
		}(i)
	}

	// Main loop for receiving messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Receive messages in batches
				result, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
					QueueUrl:            aws.String(queueURL),
					MaxNumberOfMessages: int32(batchSize),
					WaitTimeSeconds:     20, // Long polling
				})

				if err != nil {
					logger.Error("failed to receive messages", zap.Error(err))
					time.Sleep(time.Second)
					continue
				}

				// Process received messages
				for _, msg := range result.Messages {
					select {
					case <-ctx.Done():
						return
					case processMessages <- msg:
						// Message sent for processing
					}
				}
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Info("shutdown signal received, stopping workers...")
	cancel()
	close(processMessages)
	wg.Wait()
	logger.Info("shutdown complete")
}

func processMessage(ctx context.Context, logger *zap.Logger, sqsClient *sqs.Client, queueURL string, msg types.Message, maxRetries, workerID int) {
	logger.Info("processing message",
		zap.String("message_id", *msg.MessageId),
		zap.Int("worker_id", workerID))

	// Simulate processing time and potential failures
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Simulate processing time
		time.Sleep(100 * time.Millisecond)

		// Simulate random processing failures (20% chance)
		if attempt < maxRetries && time.Now().UnixNano()%5 == 0 {
			logger.Error("processing failed, retrying",
				zap.String("message_id", *msg.MessageId),
				zap.Int("attempt", attempt),
				zap.Int("worker_id", workerID))
			continue
		}

		// Delete message after successful processing
		_, err := sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(queueURL),
			ReceiptHandle: msg.ReceiptHandle,
		})

		if err != nil {
			logger.Error("failed to delete message",
				zap.String("message_id", *msg.MessageId),
				zap.Error(err),
				zap.Int("worker_id", workerID))
			continue
		}

		logger.Info("message processed and deleted successfully",
			zap.String("message_id", *msg.MessageId),
			zap.Int("attempts_needed", attempt),
			zap.Int("worker_id", workerID))
		return
	}

	logger.Error("message processing failed after max retries",
		zap.String("message_id", *msg.MessageId),
		zap.Int("max_retries", maxRetries),
		zap.Int("worker_id", workerID))
}
