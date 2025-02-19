# Build Stage (Smaller, Optimized)
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o go-redis -ldflags "-s -w" main.go  # Optimize binary size

# Runtime Stage (Lightweight)
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/go-redis .
VOLUME ["/data"]  # Define volume for AOF persistence
# Expose Redis port
EXPOSE 6379
CMD ["./go-redis", "--aof-path", "/data/database.aof"]
