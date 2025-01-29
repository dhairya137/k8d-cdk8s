# AWS SQS with KEDA Auto-scaling

This project demonstrates how to set up AWS SQS with KEDA auto-scaling in Kubernetes. It includes a producer that sends messages to an SQS queue and a consumer that processes these messages with automatic scaling based on queue length.

## Project Structure

```
sqs-scaling/
├── consumer/
│   ├── configmap.yaml
│   ├── deployment.yaml
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   └── main.go
├── producer/
│   ├── deployment.yaml
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   └── main.go
├── aws-secret.yaml
└── SQS-ScaledObject.yaml
```

## Prerequisites

1. AWS Setup:

   - An AWS account with access to SQS
   - AWS credentials with permissions to:
     - Create/Delete SQS queues
     - Send/Receive messages
     - Get queue attributes
   - SQS queue created in your AWS account

2. Kubernetes Setup:
   - Kubernetes cluster with KEDA installed
   - kubectl configured to access your cluster

## Setup Instructions

### 1. Create AWS Secret

Create a Kubernetes secret with AWS credentials:

```yaml
# aws-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-secret
type: Opaque
data:
  AWS_ACCESS_KEY_ID: <base64-encoded-access-key>
  AWS_SECRET_ACCESS_KEY: <base64-encoded-secret-key>
  AWS_REGION: <base64-encoded-region>
```

Apply the secret:

```bash
kubectl apply -f aws-secret.yaml
```

### 2. Setup Producer

The producer generates messages and sends them to an SQS queue.

Key producer configuration:

```go
const (
    queueName = "test-queue"
    batchSize = 50
)
```

Features:

- Connects to AWS SQS using environment variables
- Generates test messages
- Sends messages in batches
- Configurable message count and interval

Build and deploy:

```bash
cd producer
docker build -t your-registry/sqs-producer:v1 .
docker push your-registry/sqs-producer:v1
kubectl apply -f deployment.yaml
```

### 3. Setup Consumer

The consumer processes messages from the SQS queue.

Key consumer configuration:

```go
const (
    queueName = "test-queue"
    batchSize = 10
)
```

Features:

- Long polling for efficient message retrieval
- Batch processing capability
- Error handling with retry mechanism
- Graceful shutdown handling
- Automatic message deletion after successful processing

Build and deploy:

```bash
cd consumer
docker build -t your-registry/sqs-consumer:v1 .
docker push your-registry/sqs-consumer:v1
kubectl apply -f deployment.yaml
```

### 4. Configure KEDA Scaling

Deploy the ScaledObject for KEDA:

```yaml
# SQS-ScaledObject.yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: aws-sqs-scaledobject
  namespace: default
spec:
  scaleTargetRef:
    name: consumer-deployment
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
          policies:
            - type: Percent
              value: 100
              periodSeconds: 5
  triggers:
    - type: aws-sqs-queue
      metadata:
        queueURL: https://sqs.<region>.amazonaws.com/<account-id>/<queue-name>
        queueLength: "5"
        activationQueueLength: "3"
        awsRegion: "<region>"
        identityOwner: "pod"
```

Apply the KEDA configuration:

```bash
kubectl apply -f SQS-ScaledObject.yaml
```

## Monitoring and Testing

1. Check SQS queue:

```bash
# Get queue attributes
aws sqs get-queue-attributes \
  --queue-url <your-queue-url> \
  --attribute-names ApproximateNumberOfMessages
```

2. Watch scaling:

```bash
# Watch HPA
kubectl get hpa -w

# Watch consumer pods
kubectl get pods -l app=consumer -w
```

3. Check logs:

```bash
# Consumer logs
kubectl logs -l app=consumer

# Producer logs
kubectl logs -l app=producer
```

## Scaling Behavior

- KEDA monitors SQS queue length
- Scales up when queue length exceeds 5 messages
- Begins scaling at 3 messages (activationQueueLength)
- Scales down after 10 seconds of low activity
- Maintains between 0 and 10 replicas
- Fast scale down with 10s stabilization window

## Architecture Details

1. Message Flow:

   - Producer → AWS SQS Queue → Consumer
   - Messages stored in SQS with configurable retention
   - FIFO or Standard queue support

2. Consumer Processing:

   - Long polling for efficient message retrieval
   - Batch processing for better throughput
   - Automatic message deletion after processing
   - Graceful shutdown handling

3. Scaling Mechanism:

   - Queue length-based scaling
   - Quick scale-up for traffic spikes
   - Efficient scale-down
   - Configurable thresholds

4. High Availability:
   - AWS SQS built-in redundancy
   - Message visibility timeout
   - Dead-letter queue support
   - Kubernetes pod distribution

## Troubleshooting

1. Check AWS credentials:

```bash
kubectl describe secret aws-secret
```

2. Verify SQS access:

```bash
kubectl exec -it <consumer-pod> -- aws sqs get-queue-url --queue-name <queue-name>
```

3. Monitor consumer status:

```bash
kubectl describe pod -l app=consumer
kubectl describe hpa
```

## Notes

- Ensure proper AWS IAM permissions
- Consider queue type (Standard vs FIFO)
- Adjust visibility timeout based on processing time
- Configure dead-letter queue for failed messages
- Monitor AWS costs
- Tune KEDA scaling parameters based on workload patterns
