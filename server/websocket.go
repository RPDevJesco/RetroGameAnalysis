package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketManager handles multiple WebSocket connections
type WebSocketManager struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
}

// Client represents a WebSocket client connection
type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	manager  *WebSocketManager
	id       string
	metadata map[string]interface{}
}

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	ClientID  string                 `json:"client_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}
}

// Start starts the WebSocket manager
func (m *WebSocketManager) Start() {
	go m.run()
}

// run handles the main WebSocket manager loop
func (m *WebSocketManager) run() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client] = true
			m.mu.Unlock()

			log.Printf("WebSocket client connected: %s", client.id)

			// Send welcome message
			welcome := Message{
				Type:      "connected",
				Data:      map[string]interface{}{"message": "WebSocket connection established"},
				Timestamp: time.Now(),
			}
			client.SendMessage(welcome)

			// Notify other clients
			m.BroadcastMessage(Message{
				Type:      "client_connected",
				Data:      map[string]interface{}{"client_id": client.id},
				Timestamp: time.Now(),
			})

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				close(client.send)
			}
			m.mu.Unlock()

			log.Printf("WebSocket client disconnected: %s", client.id)

			// Notify other clients
			m.BroadcastMessage(Message{
				Type:      "client_disconnected",
				Data:      map[string]interface{}{"client_id": client.id},
				Timestamp: time.Now(),
			})

		case message := <-m.broadcast:
			m.mu.RLock()
			for client := range m.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, close connection
					close(client.send)
					delete(m.clients, client)
				}
			}
			m.mu.RUnlock()
		}
	}
}

// BroadcastMessage sends a message to all connected clients
func (m *WebSocketManager) BroadcastMessage(message Message) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v", err)
		return
	}

	select {
	case m.broadcast <- data:
	default:
		// Broadcast channel is full, skip this message
		log.Printf("Broadcast channel full, skipping message")
	}
}

// BroadcastPropertyChange sends a property change notification
func (m *WebSocketManager) BroadcastPropertyChange(propertyName string, value interface{}, metadata map[string]interface{}) {
	message := Message{
		Type: "property_changed",
		Data: map[string]interface{}{
			"property": propertyName,
			"value":    value,
		},
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
	m.BroadcastMessage(message)
}

// BroadcastMapperLoaded sends a mapper loaded notification
func (m *WebSocketManager) BroadcastMapperLoaded(mapperName string) {
	message := Message{
		Type: "mapper_loaded",
		Data: map[string]interface{}{
			"mapper": mapperName,
		},
		Timestamp: time.Now(),
	}
	m.BroadcastMessage(message)
}

// BroadcastError sends an error notification
func (m *WebSocketManager) BroadcastError(errorType, errorMessage string) {
	message := Message{
		Type: "error",
		Data: map[string]interface{}{
			"error_type": errorType,
			"message":    errorMessage,
		},
		Timestamp: time.Now(),
	}
	m.BroadcastMessage(message)
}

// GetClientCount returns the number of connected clients
func (m *WebSocketManager) GetClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// GetClientIDs returns a list of connected client IDs
func (m *WebSocketManager) GetClientIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.clients))
	for client := range m.clients {
		ids = append(ids, client.id)
	}
	return ids
}

// HandleWebSocket upgrades HTTP connection to WebSocket
func (m *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create client
	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		manager:  m,
		id:       generateClientID(),
		metadata: make(map[string]interface{}),
	}

	// Extract client metadata from headers
	if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
		client.metadata["user_agent"] = userAgent
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		client.metadata["origin"] = origin
	}
	client.metadata["remote_addr"] = r.RemoteAddr
	client.metadata["connected_at"] = time.Now()

	// Register client
	m.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// SendMessage sends a message to this specific client
func (c *Client) SendMessage(message Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	default:
		return websocket.ErrCloseSent
	}
}

// readPump handles reading messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.manager.unregister <- c
		c.conn.Close()
	}()

	// Set read deadline and pong handler
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var rawMessage json.RawMessage
		err := c.conn.ReadJSON(&rawMessage)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle client message
		c.handleMessage(rawMessage)
	}
}

// writePump handles writing messages to the WebSocket connection
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

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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

// handleMessage processes incoming messages from the client
func (c *Client) handleMessage(rawMessage json.RawMessage) {
	var message Message
	if err := json.Unmarshal(rawMessage, &message); err != nil {
		log.Printf("Error unmarshaling client message: %v", err)
		return
	}

	// Add client ID to message
	message.ClientID = c.id
	message.Timestamp = time.Now()

	// Handle different message types
	switch message.Type {
	case "ping":
		// Respond with pong
		c.SendMessage(Message{
			Type:      "pong",
			Timestamp: time.Now(),
		})

	case "subscribe":
		// Handle subscription to specific events
		c.handleSubscription(message)

	case "get_status":
		// Send current status
		c.SendMessage(Message{
			Type: "status",
			Data: map[string]interface{}{
				"client_count": c.manager.GetClientCount(),
				"client_id":    c.id,
				"uptime":       time.Since(c.metadata["connected_at"].(time.Time)).String(),
			},
			Timestamp: time.Now(),
		})

	default:
		log.Printf("Unknown message type from client %s: %s", c.id, message.Type)
	}
}

// handleSubscription processes subscription requests
func (c *Client) handleSubscription(message Message) {
	if data, ok := message.Data.(map[string]interface{}); ok {
		if events, ok := data["events"].([]interface{}); ok {
			c.metadata["subscribed_events"] = events
			log.Printf("Client %s subscribed to events: %v", c.id, events)
		}
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
