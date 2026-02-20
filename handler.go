package main

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)


func ping(args []Value) Value {
	if len(args) == 0 {
		return Value{typ: "string", str: "PONG"}
	}
	return Value{typ: "string", str: args[0].bulk}
}

var Handlers = map[string]func([]Value) Value{ // associate a command with handler funciton
	"PING":     ping,
	"GET":      get,
	"SET":      set,
	"HSET":     hset,
	"HGET":     hget,
	"HGETALL":  hgetall,
	"DEL":      del,
	"COMMAND":  command,
	"PUBLISH":  publish,
	"EXPIRE":   expire,
	"TTL":      ttl,
	"EXPIREAT": expireat,
}

// StringEntry represents a string value stored via SET, with optional
// expiration metadata. ExpiresAt is used for passive expiration checks
// in handlers like GET and DEL; keys with an ExpiresAt in the past are
// treated as non-existent.
type StringEntry struct {
	Value     string
	ExpiresAt *time.Time // nil means no expiration
}

// isExpired reports whether the given StringEntry is logically expired
// based on its ExpiresAt timestamp. A nil ExpiresAt means the entry
// does not have an expiration and is therefore not expired.
func isExpired(entry StringEntry) bool {
	if entry.ExpiresAt == nil {
		return false
	}
	now := time.Now()
	return now.After(*entry.ExpiresAt)
}

var SETs = map[string]StringEntry{}
var SETsMu = sync.RWMutex{}

func set(args []Value) Value {
	if len(args) != 2 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'set' command"}
	}
	key := args[0].bulk
	value := args[1].bulk
	SETsMu.Lock()
	SETs[key] = StringEntry{
		Value:     value,
		ExpiresAt: nil, // TTL will be set in future steps
	}
	SETsMu.Unlock()
	return Value{typ: "string", str: "OK"}
}

func get(args []Value) Value {
	if len(args) != 1 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'get' command"}
	}
	key := args[0].bulk

	// First perform a read-only lookup to see if the key exists.
	SETsMu.RLock()
	entry, ok := SETs[key]
	SETsMu.RUnlock()
	if !ok {
		return Value{typ: "null"}
	}

	// Passive expiration: if the key is expired, delete it and treat it as
	// non-existent for this GET, mirroring Redis semantics.
	if isExpired(entry) {
		SETsMu.Lock()
		if entry2, ok2 := SETs[key]; ok2 && isExpired(entry2) {
			delete(SETs, key)
		}
		SETsMu.Unlock()
		return Value{typ: "null"}
	}

	return Value{typ: "bulk", bulk: entry.Value}
}

var HSETs = map[string]map[string]string{} // map with key: string, and value: map of string-string
var HSETsMu = sync.RWMutex{}

func hset(args []Value) Value {
	if len(args) != 3 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'hset' command"}
	}
	hashKey := args[0].bulk
	key := args[1].bulk
	value := args[2].bulk
	HSETsMu.Lock()
	if _, ok := HSETs[hashKey]; !ok {
		HSETs[hashKey] = map[string]string{}
	}
	HSETs[hashKey][key] = value
	HSETsMu.Unlock()
	return Value{typ: "string", str: "OK"}
}

func hget(args []Value) Value{
	if len(args) != 2 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'hget' command"}
	}

	hash := args[0].bulk
	key := args[1].bulk

	HSETsMu.RLock()
	value, ok := HSETs[hash][key]
	HSETsMu.RUnlock()
	if !ok {
		return Value{typ: "null"}
	}
	return Value{typ: "bulk", bulk: value}
}

func hgetall(args []Value) Value{
	if len(args) != 1 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'hgetall' command"}
	}

	hashKey := args[0].bulk

	HSETsMu.RLock()
	hash, ok := HSETs[hashKey]
	HSETsMu.RUnlock()
	if !ok {
		return Value{typ: "null"}
	}

	var result []Value
	for k, v := range hash {
		result = append(result, Value{typ: "bulk", bulk: k})
		result = append(result, Value{typ: "bulk", bulk: v})
	}

	return Value{typ: "array", array: result}
}

func del(args []Value) Value {
	if len(args) != 1 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'del' command"}
	}
	key := args[0].bulk

	// String-key deletion with passive expiration semantics: expired keys
	// are treated as non-existent for this command, though we still clean
	// them up from SETs. Use a single write lock since DEL is inherently
	// a write operation.
	stringDeleted := false

	SETsMu.Lock()
	if entry, exists := SETs[key]; exists {
		if !isExpired(entry) {
			// Non-expired key: delete and count as successful deletion
			delete(SETs, key)
			stringDeleted = true
		} else {
			// Expired key: clean up but don't count as deletion
			delete(SETs, key)
		}
	}
	SETsMu.Unlock()

	// Hash-key deletion behavior is unchanged.
	existsInHSETs := false
	HSETsMu.Lock()
	if _, existsInHSETs = HSETs[key]; existsInHSETs {
		delete(HSETs, key)
	}
	HSETsMu.Unlock()

	if stringDeleted || existsInHSETs {
		return Value{typ: "string", str: "OK"}
	}
	return Value{typ: "string", str: "NOT FOUND"}
}

// expire sets a timeout on key. After the timeout, the key is automatically
// deleted. Returns 1 if timeout was set, 0 if key does not exist (or is
// already expired, in which case it is lazily deleted).
func expire(args []Value) Value {
	if len(args) != 2 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'expire' command"}
	}
	key := args[0].bulk
	seconds, err := strconv.ParseInt(args[1].bulk, 10, 64)
	if err != nil {
		return Value{typ: "error", str: "ERR value is not an integer or out of range"}
	}

	SETsMu.Lock()
	defer SETsMu.Unlock()

	entry, ok := SETs[key]
	if !ok {
		return Value{typ: "integer", num: 0}
	}
	if isExpired(entry) {
		delete(SETs, key)
		return Value{typ: "integer", num: 0}
	}

	t := time.Now().Add(time.Duration(seconds) * time.Second)
	SETs[key] = StringEntry{Value: entry.Value, ExpiresAt: &t}
	return Value{typ: "integer", num: 1}
}

// ttl returns the remaining time-to-live of key in seconds.
// Returns -2 if the key does not exist (or is expired), -1 if no expiry is
// set, or the remaining whole seconds otherwise.
func ttl(args []Value) Value {
	if len(args) != 1 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'ttl' command"}
	}
	key := args[0].bulk

	SETsMu.RLock()
	entry, ok := SETs[key]
	SETsMu.RUnlock()

	if !ok {
		return Value{typ: "integer", num: -2}
	}

	if isExpired(entry) {
		// Lock upgrade: release read lock, acquire write lock, re-check before deleting.
		SETsMu.Lock()
		if e2, ok2 := SETs[key]; ok2 && isExpired(e2) {
			delete(SETs, key)
		}
		SETsMu.Unlock()
		return Value{typ: "integer", num: -2}
	}

	if entry.ExpiresAt == nil {
		return Value{typ: "integer", num: -1}
	}

	remaining := int(time.Until(*entry.ExpiresAt).Seconds())
	return Value{typ: "integer", num: remaining}
}

// expireat sets the expiration of key to an absolute Unix timestamp (seconds).
// It is used primarily during AOF replay to restore exact expiry deadlines.
// Returns 1 if set, 0 if the key does not exist or is already expired.
func expireat(args []Value) Value {
	if len(args) != 2 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'expireat' command"}
	}
	key := args[0].bulk
	unixSec, err := strconv.ParseInt(args[1].bulk, 10, 64)
	if err != nil {
		return Value{typ: "error", str: "ERR value is not an integer or out of range"}
	}

	SETsMu.Lock()
	defer SETsMu.Unlock()

	entry, ok := SETs[key]
	if !ok {
		return Value{typ: "integer", num: 0}
	}
	if isExpired(entry) {
		delete(SETs, key)
		return Value{typ: "integer", num: 0}
	}

	t := time.Unix(unixSec, 0)
	SETs[key] = StringEntry{Value: entry.Value, ExpiresAt: &t}
	return Value{typ: "integer", num: 1}
}

func command(args []Value) Value {
	return Value{typ: "array", array: []Value{}}
}


var pubSubManager = NewPubSubManager() // Create Pub/Sub manager


func publish(args []Value) Value {
	if len(args) != 2 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'publish' command"}
	}
	channel := args[0].bulk
	message := args[1].bulk
	pubSubManager.Publish(channel, message)
	return Value{typ: "string", str: "Message published to " + channel}
}


func handleSubscription(conn net.Conn, args []Value) {
	if len(args) == 0 {
		NewWriter(conn).Write(Value{typ: "error", str: "ERR wrong number of arguments for 'subscribe' command"})
		return
	}

	quitChan := make(chan struct{}) // Quit channel for graceful disconnects

	// Subscribe the client to multiple channels
	for _, arg := range args {
		channel := arg.bulk
		pubSubManager.Subscribe(conn, channel, quitChan)
	}

	// Keep connection alive and listen for quit signal
	<-quitChan
	fmt.Println("Client disconnected:", conn.RemoteAddr())
	conn.Close() // Close connection when subscriber quits
}

