package websocket

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
	"github.com/gorilla/websocket"
)

// WebSocket connection upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// In production, you should check the origin
		return true
	},
}

// Hub manages WebSocket connections and broadcasts messages
type Hub struct {
	// Registered clients by delivery ID
	clients map[int]map[*Client]bool

	// Inbound messages from clients
	broadcast chan *LocationMessage

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread safety
	mutex sync.RWMutex
}

// LocationMessage represents a location update message
type LocationMessage struct {
	DeliveryID int             `json:"delivery_id"`
	Location   *domain.Location `json:"location"`
}

// Client represents a WebSocket client
type Client struct {
	// The WebSocket connection
	conn *websocket.Conn

	// The delivery ID this client is tracking
	deliveryID int

	// Buffered channel of outbound messages
	send chan *LocationMessage

	// Reference to the hub
	hub *Hub
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int]map[*Client]bool),
		broadcast:  make(chan *LocationMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub and handles client registration/unregistration and broadcasting
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			if h.clients[client.deliveryID] == nil {
				h.clients[client.deliveryID] = make(map[*Client]bool)
			}
			h.clients[client.deliveryID][client] = true
			log.Printf("Client registered for delivery %d. Total clients: %d", client.deliveryID, len(h.clients[client.deliveryID]))
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if clients, ok := h.clients[client.deliveryID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					log.Printf("Client unregistered from delivery %d. Remaining clients: %d", client.deliveryID, len(clients))

					// Clean up empty delivery maps
					if len(clients) == 0 {
						delete(h.clients, client.deliveryID)
					}
				}
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			if clients, ok := h.clients[message.DeliveryID]; ok {
				for client := range clients {
					select {
					case client.send <- message:
					default:
						// Client send channel is full, close the connection
						close(client.send)
						delete(clients, client)
					}
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastLocation broadcasts a location update to all clients tracking the delivery
func (h *Hub) BroadcastLocation(deliveryID int, location *domain.Location) {
	message := &LocationMessage{
		DeliveryID: deliveryID,
		Location:   location,
	}
	h.broadcast <- message
}

// HandleWebSocket handles WebSocket connections for live tracking
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract delivery ID from URL path
	// Expected path: /ws/deliveries/{id}/track
	path := strings.TrimPrefix(r.URL.Path, "/ws/deliveries/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 || parts[1] != "track" {
		http.Error(w, "Invalid WebSocket path", http.StatusBadRequest)
		return
	}

	deliveryID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "Invalid delivery ID", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create client
	client := &Client{
		conn:       conn,
		deliveryID: deliveryID,
		send:       make(chan *LocationMessage, 256),
		hub:        h,
	}

	// Register client
	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		// For now, we don't handle messages from clients
		// In the future, we could handle client-specific requests
	}
}

// writePump pumps messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("Error writing JSON to WebSocket: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
