package main

import (
	"fmt"
	"net"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", "localhost:6379")
	fmt.Println("Listening on port :6379")
	if err != nil {
		fmt.Println(err)
		return
	}

	aof, err := NewAof("database.aof")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer aof.Close()
	aof.Read(func (value Value) {
		command := strings.ToUpper(value.array[0].bulk)
		args := value.array[1:]
		handler, ok := Handlers[command]
		if !ok {
			fmt.Println("Invalid command: ", command)
			return
		}
		handler(args)
	})

	conn, err := listener.Accept()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	for {
		resp := NewResp(conn)
		value, err := resp.Read()
		if err != nil{
			fmt.Println(err)
			return
		}
		fmt.Println(value)
		if value.typ != "array" {
			fmt.Println("Invalid request, expected array")
			continue
		}
		if len(value.array) == 0 {
			fmt.Println("Invalid request, expected array length > 0")
			continue
		}

		writer := NewWriter(conn)
		command := strings.ToUpper(value.array[0].bulk)
		args := value.array[1:]
		handler, ok := Handlers[command]
		if !ok {
			fmt.Println("Invalid command: ", command)
			writer.Write(Value{typ: "string", str: ""})
			continue
		}
		if command == "SET" || command == "HSET" || command == "DEL" {
			aof.Write(value)
		}

		result := handler(args)
		writer.Write(result)
	}

	
}