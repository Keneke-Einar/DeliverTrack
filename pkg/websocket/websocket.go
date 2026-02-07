package websocket

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
	authPorts "github.com/Keneke-Einar/delivertrack/pkg/auth/ports"
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
	clients         map[int]map[*Client]bool // Registered clients by delivery ID
	customerClients map[int]map[*Client]bool // Registered clients by customer ID for notifications
	broadcast       chan *LocationMessage    // Inbound messages from clients
	customerBroadcast chan *NotificationMessage // Customer notification messages
	register        chan *Client             // Register requests from clients
	unregister      chan *Client             // Unregister requests from clients
	authService     authPorts.AuthService    // Auth service for token validation
	connectionCount int                      // Connection count for metrics
	mutex           sync.RWMutex             // Mutex for thread safety
}

// LocationMessage represents a location update message
type LocationMessage struct {
	DeliveryID int              `json:"delivery_id"`
	Location   *domain.Location `json:"location"`
}
// NotificationMessage represents a customer notification message
type NotificationMessage struct {
	CustomerID int    `json:"customer_id"`
	Type       string `json:"type"` // "status_update", "eta_update", etc.
	Message    string `json:"message"`
	Data       interface{} `json:"data,omitempty"`
}
// Client represents a WebSocket client
type Client struct {
	// The WebSocket connection
	conn *websocket.Conn

	// The delivery ID this client is tracking (for delivery tracking)
	deliveryID int

	// User information for authorization
	userID     int
	username   string
	role       string
	customerID *int
	courierID  *int

	// Client type: "delivery_tracker" or "customer_notifications"
	clientType string

	// Buffered channel of outbound messages
	send chan interface{}

	// Reference to the hub
	hub *Hub
}

// NewHub creates a new WebSocket hub
func NewHub(authService authPorts.AuthService) *Hub {
	return &Hub{
		clients:           make(map[int]map[*Client]bool),
		customerClients:   make(map[int]map[*Client]bool),
		broadcast:         make(chan *LocationMessage),
		customerBroadcast: make(chan *NotificationMessage),
		register:          make(chan *Client),
		unregister:        make(chan *Client),
		authService:       authService,
		connectionCount:   0,
	}
}

// Run starts the hub and handles client registration/unregistration and broadcasting
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			if client.clientType == "delivery_tracker" {
				if h.clients[client.deliveryID] == nil {
					h.clients[client.deliveryID] = make(map[*Client]bool)
				}
				h.clients[client.deliveryID][client] = true
				log.Printf("Delivery tracker registered for delivery %d. Total clients: %d", client.deliveryID, len(h.clients[client.deliveryID]))
			} else if client.clientType == "customer_notifications" {
				if client.customerID != nil {
					if h.customerClients[*client.customerID] == nil {
						h.customerClients[*client.customerID] = make(map[*Client]bool)
					}
					h.customerClients[*client.customerID][client] = true
					log.Printf("Customer notification client registered for customer %d. Total clients: %d", *client.customerID, len(h.customerClients[*client.customerID]))
				}
			}
			h.connectionCount++
			log.Printf("Total connections: %d", h.connectionCount)
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if client.clientType == "delivery_tracker" {
				if clients, ok := h.clients[client.deliveryID]; ok {
					if _, ok := clients[client]; ok {
						delete(clients, client)
						close(client.send)
						log.Printf("Delivery tracker unregistered from delivery %d. Remaining clients: %d", client.deliveryID, len(clients))

						// Clean up empty delivery maps
						if len(clients) == 0 {
							delete(h.clients, client.deliveryID)
						}
					}
				}
			} else if client.clientType == "customer_notifications" {
				if client.customerID != nil {
					if clients, ok := h.customerClients[*client.customerID]; ok {
						if _, ok := clients[client]; ok {
							delete(clients, client)
							close(client.send)
							log.Printf("Customer notification client unregistered from customer %d. Remaining clients: %d", *client.customerID, len(clients))

							// Clean up empty customer maps
							if len(clients) == 0 {
								delete(h.customerClients, *client.customerID)
							}
						}
					}
				}
			}
			h.connectionCount--
			log.Printf("Total connections: %d", h.connectionCount)
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

		case notification := <-h.customerBroadcast:
			h.mutex.RLock()
			if clients, ok := h.customerClients[notification.CustomerID]; ok {
				for client := range clients {
					select {
					case client.send <- notification:
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

// BroadcastCustomerNotification broadcasts a notification to all clients subscribed to the customer
func (h *Hub) BroadcastCustomerNotification(customerID int, notificationType, message string, data interface{}) {
	notification := &NotificationMessage{
		CustomerID: customerID,
		Type:       notificationType,
		Message:    message,
		Data:       data,
	}
	h.customerBroadcast <- notification
}

// GetConnectionCount returns the current number of active WebSocket connections
func (h *Hub) GetConnectionCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.connectionCount
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

	// Extract and validate JWT token from query parameters
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"unauthorized","message":"Token required in query parameter"}`, http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := h.authService.ValidateToken(r.Context(), token)
	if err != nil {
		http.Error(w, `{"error":"unauthorized","message":"Invalid or expired token"}`, http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create client with user information
	client := &Client{
		conn:       conn,
		deliveryID: deliveryID,
		userID:     claims.UserID,
		username:   claims.Username,
		role:       claims.Role,
		customerID: claims.CustomerID,
		courierID:  claims.CourierID,
		clientType: "delivery_tracker",
		send:       make(chan interface{}, 256),
		hub:        h,
	}

	log.Printf("WebSocket authenticated: user %s (%s) tracking delivery %d", claims.Username, claims.Role, deliveryID)

	// Register client
	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// HandleCustomerWebSocket handles WebSocket connections for customer notifications
func (h *Hub) HandleCustomerWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract and validate JWT token from query parameters
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"unauthorized","message":"Token required in query parameter"}`, http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := h.authService.ValidateToken(r.Context(), token)
	if err != nil {
		http.Error(w, `{"error":"unauthorized","message":"Invalid or expired token"}`, http.StatusUnauthorized)
		return
	}

	// Only customers can subscribe to notifications
	if claims.Role != "customer" {
		http.Error(w, `{"error":"forbidden","message":"Only customers can subscribe to notifications"}`, http.StatusForbidden)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create client with user information
	client := &Client{
		conn:       conn,
		deliveryID: 0, // Not tracking a specific delivery
		userID:     claims.UserID,
		username:   claims.Username,
		role:       claims.Role,
		customerID: claims.CustomerID,
		courierID:  claims.CourierID,
		clientType: "customer_notifications",
		send:       make(chan interface{}, 256),
		hub:        h,
	}

	customerIDStr := "N/A"
	if claims.CustomerID != nil {
		customerIDStr = fmt.Sprintf("%d", *claims.CustomerID)
	}
	log.Printf("Customer WebSocket authenticated: user %s (customer %s) subscribed to notifications", claims.Username, customerIDStr)

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
