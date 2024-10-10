package core

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Define your WebSocket server struct
type WsQuoteServer struct {
	clients    map[*websocket.Conn]bool // map to track clients
	broadcast  chan []byte              // channel to broadcast messages
	register   chan *websocket.Conn     // channel for new connections
	unregister chan *websocket.Conn     // channel for disconnections
}

// Initialize WsQuoteServer
func NewWsQuoteServer() *WsQuoteServer {
	return &WsQuoteServer{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Method to start the WebSocket server
func (server *WsQuoteServer) Start() {
	// Configure WebSocket upgrade
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// WebSocket handler function
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}

		// Register client
		server.register <- conn

		// Handle client messages
		go server.handleMessages(conn)
	})

	// Start listening for connections
	go server.listenClients()

	// Start broadcasting messages
	go server.broadcastMessages()

	// Start heartbeat checker
	go server.checkHeartbeat()

	// Start HTTP server
	log.Println("WebSocket server started")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// Method to handle incoming messages from clients
func (server *WsQuoteServer) handleMessages(conn *websocket.Conn) {
	defer func() {
		server.unregister <- conn
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		// Handle incoming messages here
	}
}

// Method to listen for new clients and disconnects
func (server *WsQuoteServer) listenClients() {
	for {
		select {
		case conn := <-server.register:
			server.clients[conn] = true
			log.Println("Client connected")
		case conn := <-server.unregister:
			if _, ok := server.clients[conn]; ok {
				delete(server.clients, conn)
				//close(conn)
				log.Println("Client disconnected")
			}
		}
	}
}

// Method to broadcast messages to all clients
func (server *WsQuoteServer) broadcastMessages() {
	for {
		select {
		case message := <-server.broadcast:
			for conn := range server.clients {
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Println("Write error:", err)
					return
				}
			}
		}
	}
}

// Method to check client heartbeat
func (server *WsQuoteServer) checkHeartbeat() {
	for {
		time.Sleep(10 * time.Second)
		for conn := range server.clients {
			err := conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				log.Println("Heartbeat error:", err)
				server.unregister <- conn
				conn.Close()
			}
		}
	}
}

func _main() {
	server := NewWsQuoteServer()
	server.Start()
}
