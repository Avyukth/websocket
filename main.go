package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ChatRoom struct {
	clients   map[*websocket.Conn]bool
	broadcast chan Message
	mu        sync.Mutex
}

type Message struct {
	Username string `json:"username"`
	Content  string `json:"content"`
}

func (room *ChatRoom) handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	room.mu.Lock()
	room.clients[ws] = true
	room.mu.Unlock()

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			room.mu.Lock()
			delete(room.clients, ws)
			room.mu.Unlock()
			break
		}

		room.broadcast <- msg
	}
}

func (room *ChatRoom) handleMessages() {
	for {
		msg := <-room.broadcast
		for client := range room.clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				room.mu.Lock()
				delete(room.clients, client)
				room.mu.Unlock()
			}
		}
	}
}

func main() {
	room := &ChatRoom{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan Message),
	}

	http.HandleFunc("/ws", room.handleConnections)

	go room.handleMessages()

	fmt.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
