package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", ":6379")
	fmt.Println("Listening on port :6379")
	if err != nil {
		fmt.Println(err)
		return
	}

	aof, err := NewAof("./data/database.aof")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer aof.Close()
	aof.Read(func(value Value) {
		command := strings.ToUpper(value.array[0].bulk)
		args := value.array[1:]
		handler, ok := Handlers[command]
		if !ok {
			fmt.Println("Invalid command: ", command)
			return
		}
		handler(args)
	})

	// Start background expiration after AOF replay
	config := DefaultExpirationConfig()
	expirationStats, stopExpiration := StartActiveExpiration(config, aof)
	defer stopExpiration() // Ensure graceful shutdown

	// Log stats on shutdown for observability
	defer func() {
		if expirationStats != nil {
			cycles, scanned, expired := expirationStats.Snapshot()
			fmt.Printf("Expiration stats - Cycles: %d, Scanned: %d, Expired: %d\n",
				cycles, scanned, expired)
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handleConnection(conn, aof) // Handle each connection in a goroutine
	}

}

func handleConnection(conn net.Conn, aof *Aof) {
	defer conn.Close()

	for {
		resp := NewResp(conn)
		value, err := resp.Read()
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Client disconnected:", conn.RemoteAddr())
			} else {
				fmt.Println("Error reading request:", err)
			}
			return
		}
		if value.typ != "array" || len(value.array) == 0 {
			fmt.Println("Invalid request, expected array")
			continue
		}

		writer := NewWriter(conn)
		command := strings.ToUpper(value.array[0].bulk)
		args := value.array[1:]

		if command == "SUBSCRIBE" {
			go handleSubscription(conn, args)
			continue
		}
		handler, ok := Handlers[command]
		if !ok {
			writer.Write(Value{typ: "string", str: ""})
			continue
		}

		result := handler(args)

		if (command == "SET" || command == "HSET" || command == "DEL") && result.typ != "error" {
			aof.Write(value)
		}

		if command == "EXPIRE" && result.typ == "integer" && result.num == 1 {
			key := args[0].bulk
			SETsMu.RLock()
			entry, ok := SETs[key]
			SETsMu.RUnlock()
			if ok && entry.ExpiresAt != nil {
				expireatCmd := Value{
					typ: "array",
					array: []Value{
						{typ: "bulk", bulk: "EXPIREAT"},
						{typ: "bulk", bulk: key},
						{typ: "bulk", bulk: strconv.FormatInt(entry.ExpiresAt.Unix(), 10)},
					},
				}
				aof.Write(expireatCmd)
			}
		}

		writer.Write(result)

	}
}
