# **Go-Redis: A Redis Clone in Go**  

## **Overview**  
Go-Redis is a lightweight, in-memory key-value store designed as a **Redis clone**, implemented in **Go**. It supports fundamental Redis commands, including:  
- **String operations** (`SET`, `GET`, `DEL`)  
- **Hash operations** (`HSET`, `HGET`, `HGETALL`)  
- **Persistence** using an **Append-Only File (AOF)**  
- **Pub/Sub messaging system** for real-time communication  

This project mimics Redis behavior and allows developers to experiment with a simple key-value store.

---

## **Features**  
✅ **Key-Value Storage** (`SET`, `GET`, `DEL`)  
✅ **Hash Storage** (`HSET`, `HGET`, `HGETALL`)  
✅ **Pub/Sub Messaging** (`SUBSCRIBE`, `PUBLISH`)  
✅ **RESP (Redis Serialization Protocol) Support**  
✅ **Append-Only File (AOF) Persistence**  
✅ **TCP Server** listening on `localhost:6379`  

---

## **Installation**  
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

---

## **Usage**  

### **Connecting via Redis CLI**  
Since the project runs on `localhost:6379`, you can use **Redis CLI** to interact with it:  
```sh
redis-cli -p 6379
```

### **Supported Commands**  

#### **String Operations**  
```sh
SET key value  # Stores a value  
GET key        # Retrieves a value  
DEL key        # Deletes a key  
```

#### **Hash Operations**  
```sh
HSET user:1001 name "Alice"  # Stores a field-value pair  
HGET user:1001 name          # Retrieves the field value  
HGETALL user:1001            # Gets all fields & values  
```

#### **Pub/Sub Messaging**  
```sh
SUBSCRIBE news              # Subscribe to a channel  
PUBLISH news "Hello World"  # Send a message to the channel  
```

#### **Ping Test**  
```sh
PING  # Returns PONG  
```

---

## **Code Structure**  
```
📂 go-redis
├── main.go       # TCP server, command handler
├── handler.go    # Redis command logic (SET, GET, HSET, etc.)
├── pubsub.go     # Pub/Sub messaging system
├── resp.go       # RESP protocol parsing & serialization
├── aof.go        # Append-Only File (AOF) persistence
├── go.mod        # Go module file
```

---

## **Architecture**  

### **1. RESP (Redis Serialization Protocol)**
- Implements **bulk strings**, **arrays**, **simple strings**, and **errors**.  
- Parses incoming Redis-like commands and serializes responses.  

### **2. Command Handlers**  
- `Handlers` map commands (`SET`, `GET`, `HSET`, `HGET`) to Go functions.  
- Uses `sync.RWMutex` for thread safety.  

### **3. AOF Persistence**
- Logs every write command (`SET`, `HSET`, `DEL`) to `database.aof`.  
- Uses `Sync()` every second to flush writes to disk.  
- Replays `AOF` log on startup for data recovery.  

### **4. Pub/Sub Messaging**
- Supports real-time **Publish/Subscribe (Pub/Sub)** messaging.  
- Clients can `SUBSCRIBE` to a channel and receive messages.  
- Implements a **message queue** to avoid blocking subscribers.  

---

## **Future Enhancements**
- Add support for `EXPIRE` and `INCR` commands.  
- Implement **LRU eviction** for memory management.  
- Improve performance with **event-driven architecture** (epoll/kqueue).  
- Enhance **AOF compaction** for optimized storage.  
