package main

import (
	"encoding/json"
	"fmt"
	"io"
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

	port = 9876
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

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		j, err := json.Marshal(state)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("failed to marshal state"))
			return
		}
		w.Header().Add("content-type", "application/json")
		w.WriteHeader(200)
		w.Write(j)
	} else if r.Method == http.MethodPost {
		defer r.Body.Close()
		var foundState ShapeState

		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("Invalid request body"))
			return
		}
		err = json.Unmarshal(bytes, &foundState)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("Invalid ShapeStatus supplied, cannot update shape"))
		} else {
			w.WriteHeader(201)
		}

		// Keep old fields in case we only got a partial update
		newState := state
		if foundState.Color != "" {
			newState.Color = foundState.Color
		}
		if foundState.Shape != "" {
			newState.Shape = foundState.Shape
		}
		if foundState.Size != 0 {
			newState.Size = foundState.Size
		}

		stateMutex.Lock()
		state = newState
		stateMutex.Unlock()
		sendState(state)

	} else {
		w.WriteHeader(400)
		w.Write([]byte("Unsupported Method"))
	}
}

func main() {
	// Serve static files from the current directory
	fs := http.FileServer(http.FS(FrontendFS))
	http.Handle("/", fs)

	// WebSocket endpoint
	http.HandleFunc("/ws", handleWebSocket)

	http.HandleFunc("/api/status", handleStatus)

	log.Printf("Starting server on :%d\n", port)
	log.Printf("Open https://localhost:%d in your browser", port)
	if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", port), "server.crt", "server.key", nil); err != nil {
		log.Fatal(err)
	}
}
