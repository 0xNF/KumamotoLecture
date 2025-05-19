package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Global state
var (
	state = ShapeState{
		Shape: "circle",
		Color: "blue",
		Size:  100,
	}
	stateMutex sync.RWMutex

	// Manage connected clients
	clients = struct {
		sync.RWMutex
		connections map[*websocket.Conn]bool
	}{
		connections: make(map[*websocket.Conn]bool),
	}

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}
)

// sendState broadcasts the current shape state to all connected clients
func sendState(state ShapeState) {
	// Convert state to JSON
	payload, err := json.Marshal(state)
	if err != nil {
		log.Printf("Error marshaling state: %v", err)
		return
	}

	// Lock the clients map while we iterate
	clients.Lock()
	defer clients.Unlock()

	// Send to all connected clients
	for client := range clients.connections {
		err := client.WriteMessage(websocket.TextMessage, payload)
		if err != nil {
			log.Printf("Error sending state to client: %v", err)
			// Close and remove failed client
			client.Close()
			delete(clients.connections, client)
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}

	// Register new client
	clients.Lock()
	clients.connections[conn] = true
	clients.Unlock()

	// Send current state to new client
	stateMutex.RLock()
	sendState(state)
	stateMutex.RUnlock()

	// Clean up on disconnect
	defer func() {
		clients.Lock()
		delete(clients.connections, conn)
		clients.Unlock()
		conn.Close()
	}()

	// Handle incoming messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		// Parse incoming state update
		var newState ShapeState
		if err := json.Unmarshal(message, &newState); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Update state
		stateMutex.Lock()
		state = newState
		stateMutex.Unlock()

		// Broadcast new state to all clients
		sendState(newState)
	}
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
