# NATS JetStream with KEDA Auto-scaling

This project demonstrates how to set up NATS JetStream with KEDA auto-scaling in Kubernetes. It includes a producer that generates messages and a consumer that processes them, with automatic scaling based on message lag.

## Project Structure

```
nats-jetstream/
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
├── jetstream-server.yaml
├── Nats-ScaledObject.yaml
└── README.md
```

## Setup Instructions

### 1. Deploy NATS JetStream Server

Deploy the NATS JetStream server with monitoring enabled:

Check file jetsream-server.yaml

```yaml
# jetstream-server.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nats-jetstream-deployment
  labels:
    app: nats-jetstream
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nats-jetstream
  template:
    metadata:
      labels:
        app: nats-jetstream
    spec:
      containers:
        - name: nats-jetstream
          image: nats:latest
          args: ["-js", "-m", "8222"]
---
apiVersion: v1
kind: Service
metadata:
  name: nats-jetstream
  labels:
    app: nats-jetstream
spec:
  selector:
    app: nats-jetstream
  clusterIP: None
  ports:
    - name: client
      port: 4222
    - name: cluster
      port: 6222
    - name: monitor
      port: 8222
    - name: metrics
      port: 7777
    - name: leafnodes
      port: 7422
    - name: gateways
      port: 7522
```

Deploy:

```bash
kubectl apply -f jetstream-server.yaml
```

### 2. Setup Producer

The producer generates messages and publishes them to the "input" stream.

Check file prodcuer/main.go

```go
// Key producer configuration
const (
    streamName     = "input"
    streamSubjects = "input.*"
    subjectName    = "input.created"
)
```

Build and deploy:

```bash
cd producer
docker build -t your-registry/nats-producer:v1 .
docker push your-registry/nats-producer:v1
kubectl apply -f deployment.yaml
```

### 3. Setup Consumer

The consumer processes messages from the "input" stream and sends responses to the "output" stream.

Check file consumer/main.go

```go
// Key consumer configuration
inputSub, err := js.PullSubscribe("input.created", "fission_consumer",
    nats.PullMaxWaiting(512),
    nats.BindStream("input"))
```

Build and deploy:

```bash
cd consumer
docker build -t your-registry/nats-consumer:v1 .
docker push your-registry/nats-consumer:v1
kubectl apply -f deployment.yaml
```

### 4. Configure KEDA Scaling

Deploy the ScaledObject for KEDA:

Check file Nats-ScaledObject.yaml

```yaml
# Nats-ScaledObject.yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: jetstreamtest
  namespace: default
spec:
  scaleTargetRef:
    name: consumer-deployment
  pollingInterval: 10
  cooldownPeriod: 15
  minReplicaCount: 1
  maxReplicaCount: 10
  triggers:
    - type: nats-jetstream
      metadata:
        natsServerMonitoringEndpoint: "nats-jetstream.default.svc.cluster.local:8222"
        natsServer: "nats://nats-jetstream.default.svc.cluster.local:4222"
        stream: "input"
        subject: "input.created"
        consumer: "fission_consumer" #Make sure we have this same name in consumer code.
        account: "$G"
        lagThreshold: "5"
        activationLagThreshold: "3"
```

Apply the KEDA configuration:

```bash
kubectl apply -f Nats-ScaledObject.yaml
```

## Monitoring and Testing

1. Check NATS JetStream status:

```bash
kubectl port-forward svc/nats-jetstream 8222:8222
curl http://localhost:8222/jsz
```

2. Monitor scaling:

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

- KEDA monitors message lag in the "input" stream
- Scales up when lag exceeds 5 messages
- Begins scaling at 3 messages (activationLagThreshold)
- Scales down after 15 seconds of low activity (cooldownPeriod)
- Maintains between 1 and 10 replicas (min/maxReplicaCount)

## Troubleshooting

1. Check NATS connectivity:

```bash
kubectl exec -it <consumer-pod> -- nc -zv nats-jetstream 4222
```

2. Verify stream creation:

```bash
curl http://localhost:8222/jsz?streams=1
```

3. Monitor consumer status:

```bash
kubectl describe pod -l app=consumer
kubectl describe hpa
```

## Notes

- Ensure NATS monitoring endpoint is accessible
- Consumer name must match between ScaledObject and consumer code
- Messages must be properly acknowledged to prevent reprocessing
- KEDA requires proper permissions to access NATS monitoring

## Cleanup

kubectl delete -f consumer/deployment.yaml
kubectl delete -f consumer/configmap.yaml
kubectl delete -f producer/deployment.yaml
kubectl delete -f Nats-ScaledObject.yaml
