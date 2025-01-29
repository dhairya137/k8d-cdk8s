package main

import (
	"context"
	"log"
	"os"
	"strconv"
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

	// Get message count from environment
	count, err := strconv.Atoi(os.Getenv("COUNT"))
	if err != nil {
		logger.Error("invalid count provided, using default", zap.Error(err))
		count = 500 // Default to 500 messages
	}

	// Get interval from environment (in milliseconds)
	interval, err := strconv.Atoi(os.Getenv("INTERVAL"))
	if err != nil {
		logger.Error("invalid interval provided, using default", zap.Error(err))
		interval = 500 // Default to 500ms
	}

	// Continuously produce messages
	messageID := 1
	for {
		messages := make([]types.SendMessageBatchRequestEntry, 0, batchSize)
		
		// Create batch of messages
		for i := 0; i < batchSize && messageID <= count; i++ {
			id := strconv.Itoa(messageID)
			messages = append(messages, types.SendMessageBatchRequestEntry{
				Id:          aws.String(id), // Required: unique ID for each message in batch
				MessageBody: aws.String("Test" + id),
			})
			messageID++
		}

		// Reset counter if we've reached count
		if messageID > count {
			messageID = 1
		}

		// Send batch of messages
		if len(messages) > 0 {
			input := &sqs.SendMessageBatchInput{
				QueueUrl: aws.String(queueURL),
				Entries:  messages,
			}

			result, err := sqsClient.SendMessageBatch(context.Background(), input)
			if err != nil {
				logger.Error("failed to send messages",
					zap.Error(err))
				time.Sleep(time.Second) // Wait before retrying
				continue
			}

			// Log successful and failed messages
			if len(result.Successful) > 0 {
				logger.Info("messages sent successfully",
					zap.Int("count", len(result.Successful)))
			}
			if len(result.Failed) > 0 {
				logger.Error("some messages failed to send",
					zap.Int("count", len(result.Failed)))
			}
		}

		// Get current queue attributes
		attribResult, err := sqsClient.GetQueueAttributes(context.Background(), &sqs.GetQueueAttributesInput{
			QueueUrl: aws.String(queueURL),
			AttributeNames: []types.QueueAttributeName{
				types.QueueAttributeNameApproximateNumberOfMessages,
			},
		})
		if err != nil {
			logger.Error("failed to get queue attributes", zap.Error(err))
		} else {
			if messages, ok := attribResult.Attributes["ApproximateNumberOfMessages"]; ok {
				if count, err := strconv.Atoi(messages); err == nil {
					logger.Info("current queue length", zap.Int("length", count))
				}
			}
		}

		// Wait before next batch
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}
