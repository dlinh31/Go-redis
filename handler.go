package main

import (
	"sync"
)


func ping(args []Value) Value {
	if len(args) == 0 {
		return Value{typ: "string", str: "PONG"}
	}
	return Value{typ: "string", str: args[0].bulk}
}

var Handlers = map[string]func([]Value) Value{ // associate a command with handler funciton
	"PING": ping,
	"GET": get,
	"SET": set,
	"HSET": hset,
	"HGET": hget,
	"HGETALL": hgetall,
	"DEL": del,
	"COMMAND": command,
	
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
    SETsMu.RLock()
    defer SETsMu.RUnlock()
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

func del(args []Value) Value {
    if len(args) != 1 {
        return Value{typ: "error", str: "ERR wrong number of arguments for 'del' command"}
    }
    key := args[0].bulk

    existsInSETs := false
    SETsMu.RLock()
    _, existsInSETs = SETs[key]
    SETsMu.RUnlock()
    if existsInSETs {
        SETsMu.Lock()
        delete(SETs, key)
        SETsMu.Unlock()
    }

    existsInHSETs := false
    HSETsMu.Lock()
    _, existsInHSETs = HSETs[key]
    if existsInHSETs {
        delete(HSETs, key)
    }
    HSETsMu.Unlock()

    if existsInSETs || existsInHSETs {
        return Value{typ: "string", str: "OK"}
    }
    return Value{typ: "string", str: "NOT FOUND"}
}

func command(args []Value) Value {
	return Value{typ: "array", array: []Value{}} 
}
