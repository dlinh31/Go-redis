package main

import "sync"

// var Handlers = make(map[string]func([]Value) Value)

func ping(args []Value) Value {
	if len(args) == 0 {
		return Value{typ: "string", str: "PONG"}
	}
	return Value{typ: "string", str: args[0].bulk}
}

var Handlers = map[string]func([]Value) Value{
	"PING": ping,
	"GET": get,
	"SET": set,
	"HSET": hset,
	"HGET": hget,
	"HGETALL": hgetall,
}

var SETs = map[string]string{}
var SETsMu = sync.RWMutex{}

func set(args []Value) Value {
	if len(args) != 2 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'set' command"}
	}
	key := args[0].bulk
	value := args[1].bulk
	SETsMu.Lock()
	SETs[key] = value
	SETsMu.Unlock()
	return Value{typ: "string", str: "OK"}
}

func get(args []Value) Value {
	if len(args) != 1 {
		return Value{typ: "error", str: "ERR wrong number of arguments for 'get' command"}
	}
	key := args[0].bulk
	SETsMu.Lock()
	value, ok := SETs[key]
	if !ok {
		return Value{typ: "null"}
	}
	return Value{typ: "bulk", bulk: value}
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