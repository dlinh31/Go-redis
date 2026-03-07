# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o go-redis .

# Run stage
FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/go-redis .

RUN mkdir -p /app/data

EXPOSE 6379

CMD ["./go-redis"]
