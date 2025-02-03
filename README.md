# Go-Redis: A Redis Clone in Go

## Overview
Go-Redis is a lightweight, in-memory key-value store designed as a Redis clone, implemented in Go. It supports fundamental Redis commands, including string operations (`SET`, `GET`), hash operations (`HSET`, `HGET`, `HGETALL`), and persistence using an Append-Only File (AOF) mechanism.

## Features
- Supports **key-value storage** (`SET`, `GET`)
- Implements **hash storage** (`HSET`, `HGET`, `HGETALL`)
- **RESP (Redis Serialization Protocol)** parsing and serialization
- **Append-Only File (AOF) Persistence**
- **TCP Server** listens on port `6379` (default Redis port)

## Installation
Ensure you have **Go 1.23.4** installed.

Clone the repository:
```sh
git clone https://github.com/dlinh31/go-redis.git
cd go-redis
```

Run the Redis clone:
```sh
go run main.go
```

## Usage

### Connecting via Redis CLI
Since the project runs on `localhost:6379`, you can use **Redis CLI** to interact with it:
```sh
redis-cli -p 6379
```

### Supported Commands
#### **String Operations**
```sh
SET key value  # Stores a value
GET key        # Retrieves a value
```
#### **Hash Operations**
```sh
HSET user:1001 name "Alice"  # Stores a field-value pair
HGET user:1001 name          # Retrieves the field value
HGETALL user:1001            # Gets all fields & values
```
#### **Ping Test**
```sh
PING           # Returns PONG
```

## Code Structure
```
📂 go-redis
├── main.go       # TCP server, command handler
├── handler.go    # Redis command logic (SET, GET, HSET, HGET, etc.)
├── resp.go       # RESP protocol parsing & serialization
├── aof.go        # Append-Only File (AOF) persistence
├── go.mod        # Go module file
```

## Architecture
### **1. RESP (Redis Serialization Protocol)**
- Implements **bulk strings**, **arrays**, **simple strings**, and **errors**
- Parses incoming Redis-like commands and serializes responses

### **2. Command Handlers**
- `Handlers` map commands (`SET`, `GET`, `HSET`, `HGET`) to Go functions
- Uses `sync.RWMutex` for thread safety

### **3. AOF Persistence**
- Logs every write command (`SET`, `HSET`) to `database.aof`
- Uses `Sync()` every second to flush writes to disk
- Replays `AOF` log on startup for data recovery

## Future Enhancements
- Add support for `DEL`, `EXPIRE`, and `INCR` commands
- Implement LRU eviction for memory management
- Improve performance with event-driven architecture (epoll/kqueue)
