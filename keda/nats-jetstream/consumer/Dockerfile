FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o consumer

# Create final lightweight image
FROM alpine:3.14
WORKDIR /app
COPY --from=builder /app/consumer .

CMD ["./consumer"]