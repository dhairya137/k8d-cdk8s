package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

const (
	batchCount = 10
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}

	// Connect to NATS
	host := os.Getenv("NATS_SERVER")
	if host == "" {
		logger.Fatal("received empty host field")
	}
	nc, _ := nats.Connect(host)
	js, err := nc.JetStream()
	if err != nil {
		logger.Fatal("err: ", zap.Error(err))
	}

	// Subscribe to input stream for processing messages
	inputSub, err := js.PullSubscribe("input.created", "fission_consumer",
		nats.PullMaxWaiting(512),  // Changed from nats.Pull()
		nats.BindStream("input"))
	if err != nil {
		logger.Fatal("error subscribing to input stream: ", zap.Error(err))
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// Main processing loop
	for {
		select {
		case <-signalChan:
			logger.Info("Received shutdown signal")
			err = inputSub.Unsubscribe()
			if err != nil {
				logger.Error("error unsubscribing: ", zap.Error(err))
			}
			return
		default:
			// Fetch messages from input stream
			msgs, err := inputSub.Fetch(batchCount, nats.Context(ctx))
			if err != nil {
				if err != nats.ErrTimeout {
					logger.Error("error fetching messages: ", zap.Error(err))
				}
				continue
			}

			// Process messages
			for _, msg := range msgs {
				// Log received message
				logger.Info("Received message", zap.String("data", string(msg.Data)))

				// Process the message (add your processing logic here)
				processedData := "Processed: " + string(msg.Data)

				// Try to publish to response topic
				_, err := js.Publish("output.response-topic", []byte(processedData))
				if err != nil {
					// If publishing fails, send to error topic
					logger.Error("error publishing response", zap.Error(err))
					_, err = js.Publish("output.error-topic", []byte(fmt.Sprintf("Error processing: %s", string(msg.Data))))
					if err != nil {
						logger.Error("error publishing to error topic", zap.Error(err))
					}
				}

				// Acknowledge the message
				err = msg.Ack()
				if err != nil {
					logger.Error("error acknowledging message", zap.Error(err))
				}
			}
		}
	}
}