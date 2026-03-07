.PHONY: run build fmt lint check

run:
	go run main.go

build:
	go build -o go-redis .

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

check: fmt lint
