package visualization

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Server handles WebSocket connections and broadcasts events
type Server struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan Event
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.Mutex
}

// NewServer creates a new visualization server
func NewServer() *Server {
	return &Server{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan Event),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Start begins the server on the specified port
func (s *Server) Start(port int) {
	// Serve static files
	fs := http.FileServer(http.Dir("visualization/static"))
	http.Handle("/", fs)

	// Handle WebSocket connections
	http.HandleFunc("/ws", s.handleWebSocket)

	// Start the broadcast handler
	go s.handleBroadcast()

	// Start the server
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting visualization server on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	s.register <- conn

	// Handle client disconnection
	defer func() {
		s.unregister <- conn
		conn.Close()
	}()

	// Read messages (not used currently, but required for WebSocket)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (s *Server) handleBroadcast() {
	for {
		select {
		case client := <-s.register:
			s.mutex.Lock()
			s.clients[client] = true
			s.mutex.Unlock()

		case client := <-s.unregister:
			s.mutex.Lock()
			delete(s.clients, client)
			s.mutex.Unlock()

		case event := <-s.broadcast:
			s.mutex.Lock()
			for client := range s.clients {
				err := client.WriteJSON(event)
				if err != nil {
					log.Printf("Error broadcasting to client: %v", err)
					client.Close()
					delete(s.clients, client)
				}
			}
			s.mutex.Unlock()
		}
	}
}

// BroadcastEvent sends an event to all connected clients
func (s *Server) BroadcastEvent(event Event) {
	s.broadcast <- event
}
