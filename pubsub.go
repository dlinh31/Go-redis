package main

import (
	"fmt"
	"net"
	"sync"
)

type PubSubManager struct {
	mu          sync.RWMutex
	subscribers map[string]map[net.Conn]chan string
	quitChannels map[net.Conn]chan struct{} // ✅ Quit channels
}


func NewPubSubManager() *PubSubManager {
	return &PubSubManager{
		subscribers: make(map[string]map[net.Conn]chan string),
		quitChannels: make(map[net.Conn]chan struct{}), // Track quit channels
	}
}

func (ps *PubSubManager) Subscribe(conn net.Conn, channel string, quitChan chan struct{}) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, exists := ps.subscribers[channel]; !exists {
		ps.subscribers[channel] = make(map[net.Conn]chan string)
	}

	messageChan := make(chan string, 100) // Buffered message queue
	ps.subscribers[channel][conn] = messageChan
	ps.quitChannels[conn] = quitChan

	// Start listening for messages
	go listenForMessages(ps, conn, channel, messageChan, quitChan)

	// Send a "subscribe" message back to client
	writer := NewWriter(conn)
	subMsg := Value{
		typ: "array",
		array: []Value{
			{typ: "bulk", bulk: "subscribe"},   // Redis: "subscribe"
			{typ: "bulk", bulk: channel},       // Channel name
			{typ: "bulk", bulk: "1"},          // Hardcode "1" or count total channels
		},
	}
	_ = writer.Write(subMsg) // Ignore error for brevity

	fmt.Println(conn.RemoteAddr(), "subscribed to", channel)
}

func (ps *PubSubManager) Unsubscribe(conn net.Conn, channel string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if channel == "" {
		// Unsubscribe from all channels
		for ch := range ps.subscribers {
			delete(ps.subscribers[ch], conn)
			if len(ps.subscribers[ch]) == 0 {
				delete(ps.subscribers, ch)
			}
		}
	} else {
		// Unsubscribe from one channel
		if subs, exists := ps.subscribers[channel]; exists {
			delete(subs, conn)
			if len(subs) == 0 {
				delete(ps.subscribers, channel)
			}
		}
	}

	// ✅ Close quitChan to notify Goroutine to exit
	if quitChan, exists := ps.quitChannels[conn]; exists {
		close(quitChan)
		delete(ps.quitChannels, conn)
	}

	fmt.Println(conn.RemoteAddr(), "unsubscribed from", channel)
}


func (ps *PubSubManager) Publish(channel, message string) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for conn, msgChan := range ps.subscribers[channel] {
		select {
		case msgChan <- message: // Send message if channel isn't full
		default: // Avoid blocking
			fmt.Println("Subscriber queue full, dropping message:", conn.RemoteAddr())
		}
	}
}



func listenForMessages(
	ps *PubSubManager,
	conn net.Conn,
	channel string,
	msgChan chan string,
	quitChan chan struct{},
) {
	writer := NewWriter(conn)

	for {
		select {
		case msg := <-msgChan:
			// Build the 3-element array: ["message", channel, msg]
			pubsubMsg := Value{
				typ: "array",
				array: []Value{
					{typ: "bulk", bulk: "message"},   // Redis: "message"
					{typ: "bulk", bulk: channel},     // The channel name
					{typ: "bulk", bulk: msg},         // The actual message
				},
			}

			// Write the array to the client
			err := writer.Write(pubsubMsg)
			if err != nil {
				fmt.Println("Error writing to client, closing:", conn.RemoteAddr())
				ps.Unsubscribe(conn, "")
				return
			}

		case <-quitChan:
			fmt.Println("Client unsubscribed:", conn.RemoteAddr())
			return
		}
	}
}
