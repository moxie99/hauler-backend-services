package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// In production, restrict to your frontend domain
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	UserID   uint
	Conn     *websocket.Conn
	Send     chan []byte
	OrderIDs []uint // Orders this client is tracking
}

// Hub manages WebSocket clients and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("[WebSocket] Client %s connected (user %d)", client.ID, client.UserID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()
			log.Printf("[WebSocket] Client %s disconnected", client.ID)

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastToOrder broadcasts a message to all clients tracking a specific order
func (h *Hub) BroadcastToOrder(orderID uint, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		for _, trackedOrderID := range client.OrderIDs {
			if trackedOrderID == orderID {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
				break
			}
		}
	}
}

// Global hub instance
var wsHub *Hub

// InitWebSocketHub initializes the WebSocket hub
func InitWebSocketHub() {
	wsHub = NewHub()
	go wsHub.Run()
	log.Println("[WebSocket] Hub started")
}

// BroadcastOrderUpdate broadcasts order updates to connected clients
func BroadcastOrderUpdate(orderID uint, eventType string, data interface{}) {
	if wsHub == nil {
		return
	}

	message := map[string]interface{}{
		"type":      eventType,
		"order_id":  orderID,
		"data":      data,
		"timestamp": time.Now(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("[WebSocket] Error marshaling message: %v", err)
		return
	}

	wsHub.BroadcastToOrder(orderID, messageBytes)
}

// HandleWebSocket handles WebSocket connections for real-time order tracking
func HandleWebSocket(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// Get user ID for authorization
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Verify user has access to this order
	var order Order
	if err := DB.Where("id = ?", uint(orderID)).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Check if user is the customer or assigned driver
	if order.UserID != userID && (order.DriverID == nil || *order.DriverID != userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v", err)
		return
	}

	// Create client
	client := &Client{
		ID:       generateClientID(),
		UserID:   userID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		OrderIDs: []uint{uint(orderID)},
	}

	// Register client
	wsHub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// writePump handles sending messages to the client
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[WebSocket] Write error for client %s: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[WebSocket] Ping error for client %s: %v", c.ID, err)
				return
			}
		}
	}
}

// readPump handles reading messages from the client (mostly for ping/pong)
func (c *Client) readPump() {
	defer func() {
		wsHub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Read error for client %s: %v", c.ID, err)
			}
			break
		}
	}
}

// HandleSSE handles Server-Sent Events for real-time order tracking
func HandleSSE(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// Get user ID for authorization
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Verify user has access to this order
	var order Order
	if err := DB.Where("id = ?", uint(orderID)).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Check if user is the customer or assigned driver
	if order.UserID != userID && (order.DriverID == nil || *order.DriverID != userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Create a channel for this client
	clientChan := make(chan []byte, 10)

	// Create a temporary client for broadcasting
	tempClient := &Client{
		ID:       "sse-" + generateClientID(),
		UserID:   userID,
		Send:     clientChan,
		OrderIDs: []uint{uint(orderID)},
	}

	// Register temporary client
	wsHub.register <- tempClient

	// Clean up on disconnect
	defer func() {
		wsHub.unregister <- tempClient
		close(clientChan)
	}()

	// Send initial connection message
	c.SSEvent("connected", map[string]interface{}{
		"order_id": orderID,
		"status":   order.Status,
		"message":  "Connected to real-time tracking",
	})

	// Listen for messages
	c.Stream(func(w io.Writer) bool {
		select {
		case message := <-clientChan:
			var eventData map[string]interface{}
			if err := json.Unmarshal(message, &eventData); err != nil {
				return false
			}

			eventType := "update"
			if t, ok := eventData["type"].(string); ok {
				eventType = t
			}

			c.SSEvent(eventType, eventData)
			return true

		case <-time.After(30 * time.Second):
			// Send heartbeat
			c.SSEvent("heartbeat", map[string]interface{}{
				"timestamp": time.Now(),
			})
			return true
		}
	})
}
