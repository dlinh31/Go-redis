# Build Stage (Compiling Go-Redis)
FROM golang:1.23.4-alpine AS builder  
WORKDIR /app

# Copy module files first (better caching)
COPY go.mod ./
RUN go mod tidy || true  # Ignore errors if go.sum isn't needed

# Copy all Go source files
COPY . .  

# ✅ Build the entire module (not just main.go)
RUN go build -o go-redis .  

# Runtime Stage (Final Lightweight Image)
FROM alpine:latest
WORKDIR /app

# ✅ Ensure /data directory exists inside the container
RUN mkdir -p /data  

COPY --from=builder /app/go-redis .
VOLUME ["/data"]  # Define volume for AOF persistence
EXPOSE 6379

# ✅ Ensure database.aof exists before running
CMD ["sh", "-c", "touch /data/database.aof && ./go-redis --aof-path /data/database.aof"]
