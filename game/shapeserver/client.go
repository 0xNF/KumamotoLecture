package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
}

type ClientManager struct {
	connections map[*Client]bool
	sync.RWMutex
}

type ShapeState struct {
	Shape string `json:"shape"`
	Color string `json:"color"`
	Size  int    `json:"size"`
}
