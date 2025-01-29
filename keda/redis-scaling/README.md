# Redis with KEDA Auto-scaling

This project demonstrates how to set up Redis with KEDA auto-scaling in Kubernetes. It includes a producer that generates messages and pushes them to a Redis list, and a consumer that processes these messages with automatic scaling based on list length.

## Project Structure

```
redis-scaling/
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
├── redis-server.yaml
├── Redis-ScaledObject.yaml
└── README.md
```

## Setup Instructions

### 1. Deploy Redis Server

Deploy Redis with persistence enabled:

```yaml
# redis-server.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis-deployment
  labels:
    app: redis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
        - name: redis
          image: redis:latest
          args: ["--appendonly", "yes"] # Enable persistence
          ports:
            - containerPort: 6379
---
apiVersion: v1
kind: Service
metadata:
  name: redis-service
  labels:
    app: redis
spec:
  selector:
    app: redis
  ports:
    - port: 6379
      targetPort: 6379
```

Deploy:

```bash
kubectl apply -f redis-server.yaml
```

### 2. Setup Producer

The producer generates messages and pushes them to a Redis list named "workqueue".

Key producer configuration:

```go
const (
    redisListKey = "workqueue"
)
```

Features:

- Connects to Redis using environment variables
- Generates test messages
- Pushes messages to Redis list using RPUSH
- Configurable message count and interval

Build and deploy:

```bash
cd producer
docker build -t your-registry/redis-producer:v1 .
docker push your-registry/redis-producer:v1
kubectl apply -f deployment.yaml
```

### 3. Setup Consumer

The consumer processes messages from the Redis list using BLPOP.

Key consumer configuration:

```go
const (
    redisListKey = "workqueue"
    batchSize    = 10
)
```

Features:

- Blocking pop operation (BLPOP) for efficient polling
- Batch processing capability
- Error handling with retry mechanism
- Graceful shutdown handling
- Metrics for monitoring

Build and deploy:

```bash
cd consumer
docker build -t your-registry/redis-consumer:v1 .
docker push your-registry/redis-consumer:v1
kubectl apply -f deployment.yaml
```

### 4. Configure KEDA Scaling

Deploy the ScaledObject for KEDA:

```yaml
# Redis-ScaledObject.yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: redis-scaledobject
  namespace: default
spec:
  scaleTargetRef:
    name: consumer-deployment
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: 0 # Changed from 1 to allow scaling to zero
  maxReplicaCount: 10
  triggers:
    - type: redis
      metadata:
        address: redis-service.default.svc.cluster.local:6379
        listName: workqueue
        listLength: "5"
        activationListLength: "3"
```

Apply the KEDA configuration:

```bash
kubectl apply -f Redis-ScaledObject.yaml
```

## Monitoring and Testing

### Using Redis Insight

To connect Redis Insight to your Redis instance:

1. Set up port forwarding to Redis service:

```bash
kubectl port-forward svc/redis-service 6379:6379
```

2. Connect Redis Insight:

   - Open Redis Insight
   - Add new Redis Database connection
   - Use the following settings:
     - Host: localhost
     - Port: 6379
     - Name: K8s Redis
     - (No password required)

3. After connecting, you can:
   - View the "workqueue" list in the Browser section
   - Monitor list length in real-time
   - View individual messages in the queue
   - Execute Redis commands directly

### Other Monitoring Options

1. Check Redis status:

```bash
kubectl exec -it deployment/redis-deployment -- redis-cli INFO
```

2. Monitor list length:

```bash
kubectl exec -it deployment/redis-deployment -- redis-cli LLEN workqueue
```

3. Watch scaling:

```bash
# Watch HPA
kubectl get hpa -w

# Watch consumer pods
kubectl get pods -l app=consumer -w
```

4. Check logs:

```bash
# Consumer logs
kubectl logs -l app=consumer

# Producer logs
kubectl logs -l app=producer
```

## For Scaling Down

1. Scale down the producer to 0:

   `kubectl scale deployment producer-deployment --replicas=0`

2. Delete the Redis key:

   `kubectl exec -it deployment/redis-deployment -- redis-cli DEL workqueue`

3. Monitor the pods:

   `kubectl get pods -w`

- For scaling down I have seen that it might take 5 minutes.

## Scaling Behavior

- KEDA monitors Redis list length
- Scales up when list length exceeds 5 messages
- Begins scaling at 3 messages (activationListLength)
- Scales down after 30 seconds of low activity
- Maintains between 1 and 10 replicas

## Architecture Details

1. Message Flow:

   - Producer → Redis List → Consumer
   - Messages stored in Redis list "workqueue"
   - FIFO (First In, First Out) processing

2. Consumer Processing:

   - Blocking pop operations for efficiency
   - Batch processing for better throughput
   - Automatic retries on failures
   - Graceful shutdown handling

3. Scaling Mechanism:

   - List length-based scaling
   - Quick scale-up for traffic spikes
   - Gradual scale-down for stability
   - Configurable thresholds

4. High Availability:
   - Redis persistence enabled
   - Consumer retry mechanism
   - Graceful shutdown handling
   - Kubernetes service discovery

## Troubleshooting

1. Check Redis connectivity:

```bash
kubectl exec -it <consumer-pod> -- nc -zv redis-service 6379
```

2. Verify list existence:

```bash
kubectl exec -it deployment/redis-deployment -- redis-cli EXISTS workqueue
```

3. Monitor consumer status:

```bash
kubectl describe pod -l app=consumer
kubectl describe hpa
```

## Notes

- Ensure Redis has enough memory for your workload
- Monitor Redis memory usage
- Consider Redis persistence requirements
- Adjust batch size based on message processing requirements
- Tune KEDA scaling parameters based on workload patterns
