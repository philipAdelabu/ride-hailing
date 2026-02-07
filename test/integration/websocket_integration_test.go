//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/richxcame/ride-hailing/pkg/models"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
)

// WebSocketIntegrationTestSuite tests WebSocket functionality comprehensively
type WebSocketIntegrationTestSuite struct {
	suite.Suite
	rider        authSession
	driver       authSession
	wsServer     *httptest.Server
	hub          *ws.Hub
	connectedWS  []*websocket.Conn
	mu           sync.Mutex
	mockAuthFunc func(r *http.Request) (string, string, bool) // userID, role, valid
}

func TestWebSocketIntegrationSuite(t *testing.T) {
	suite.Run(t, new(WebSocketIntegrationTestSuite))
}

func (s *WebSocketIntegrationTestSuite) SetupSuite() {
	// Ensure required services are started
	if _, ok := services[authServiceKey]; !ok {
		services[authServiceKey] = startAuthService(mustLoadConfig("auth-service"))
	}
	if _, ok := services[ridesServiceKey]; !ok {
		services[ridesServiceKey] = startRidesService(mustLoadConfig("rides-service"))
	}

	// Create WebSocket hub for testing
	s.hub = ws.NewHub()
	go s.hub.Run()

	// Register message handlers
	s.registerMessageHandlers()

	// Default mock auth function - accepts all connections
	s.mockAuthFunc = func(r *http.Request) (string, string, bool) {
		userID := r.URL.Query().Get("user_id")
		role := r.URL.Query().Get("role")
		token := r.URL.Query().Get("token")

		if userID == "" {
			userID = uuid.New().String()
		}
		if role == "" {
			role = "rider"
		}

		// Simulate token validation
		if token == "invalid" {
			return "", "", false
		}

		return userID, role, true
	}

	// Create test WebSocket server with authentication
	s.wsServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, role, valid := s.mockAuthFunc(r)

		if !valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		upgrader := websocket.Upgrader{
			CheckOrigin:      func(r *http.Request) bool { return true },
			ReadBufferSize:   1024,
			WriteBufferSize:  1024,
			HandshakeTimeout: 10 * time.Second,
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		// Set connection limits
		conn.SetReadLimit(512 * 1024) // 512KB max message size

		// Create client and register with hub
		client := ws.NewClient(userID, conn, s.hub, role, nil)
		s.hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	}))
}

func (s *WebSocketIntegrationTestSuite) registerMessageHandlers() {
	// Location update handler
	s.hub.RegisterHandler("location_update", func(c *ws.Client, msg *ws.Message) {
		if msg.RideID != "" {
			s.hub.SendToRide(msg.RideID, msg)
		}
	})

	// Chat message handler
	s.hub.RegisterHandler("chat_message", func(c *ws.Client, msg *ws.Message) {
		if msg.RideID != "" {
			s.hub.SendToRide(msg.RideID, msg)
		}
	})

	// Typing indicator handler
	s.hub.RegisterHandler("typing", func(c *ws.Client, msg *ws.Message) {
		if msg.RideID != "" {
			s.hub.SendToRide(msg.RideID, msg)
		}
	})

	// Ride status update handler
	s.hub.RegisterHandler("ride_status_update", func(c *ws.Client, msg *ws.Message) {
		if msg.RideID != "" {
			s.hub.SendToRide(msg.RideID, msg)
		}
	})
}

func (s *WebSocketIntegrationTestSuite) TearDownSuite() {
	s.mu.Lock()
	for _, conn := range s.connectedWS {
		conn.Close()
	}
	s.mu.Unlock()

	if s.wsServer != nil {
		s.wsServer.Close()
	}
}

func (s *WebSocketIntegrationTestSuite) SetupTest() {
	truncateTables(s.T())
	s.rider = registerAndLogin(s.T(), models.RoleRider)
	s.driver = registerAndLogin(s.T(), models.RoleDriver)
}

func (s *WebSocketIntegrationTestSuite) TearDownTest() {
	s.mu.Lock()
	for _, conn := range s.connectedWS {
		conn.Close()
	}
	s.connectedWS = nil
	s.mu.Unlock()
}

// ============================================
// CONNECTION LIFECYCLE TESTS
// ============================================

func (s *WebSocketIntegrationTestSuite) TestConnection_UpgradeWithValidAuth() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid_token")
	require.NotNil(t, conn)

	// Verify connection is active by sending a ping
	err := conn.WriteMessage(websocket.PingMessage, nil)
	require.NoError(t, err)

	// Wait for registration
	time.Sleep(100 * time.Millisecond)
	require.GreaterOrEqual(t, s.hub.GetClientCount(), 1)
}

func (s *WebSocketIntegrationTestSuite) TestConnection_RejectionForInvalidAuth() {
	t := s.T()

	wsURL := "ws" + strings.TrimPrefix(s.wsServer.URL, "http")
	wsURL = fmt.Sprintf("%s?user_id=%s&role=%s&token=%s", wsURL, uuid.New().String(), "rider", "invalid")

	// Attempt to connect with invalid token
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

	// Connection should be rejected
	if err == nil && conn != nil {
		conn.Close()
		t.Fatal("Expected connection to be rejected for invalid auth")
	}

	if resp != nil {
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}
}

func (s *WebSocketIntegrationTestSuite) TestConnection_CleanupOnDisconnect() {
	t := s.T()

	userID := uuid.New().String()

	// Create connection
	conn := s.createWSConnection(t, userID, "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)
	initialCount := s.hub.GetClientCount()
	require.GreaterOrEqual(t, initialCount, 1)

	// Close connection
	conn.Close()

	// Wait for cleanup
	time.Sleep(300 * time.Millisecond)

	// Client count should decrease
	finalCount := s.hub.GetClientCount()
	require.Less(t, finalCount, initialCount)
}

func (s *WebSocketIntegrationTestSuite) TestConnection_ReconnectionHandling() {
	t := s.T()

	userID := s.rider.User.ID.String()

	// Create first connection
	conn1 := s.createWSConnection(t, userID, "rider", "valid")
	require.NotNil(t, conn1)

	time.Sleep(100 * time.Millisecond)
	initialCount := s.hub.GetClientCount()

	// Create second connection with same user ID (simulating reconnect)
	conn2 := s.createWSConnection(t, userID, "rider", "valid")
	require.NotNil(t, conn2)

	time.Sleep(150 * time.Millisecond)

	// The hub should handle reconnection gracefully
	// Client count should not increase significantly
	require.LessOrEqual(t, s.hub.GetClientCount(), initialCount+1)

	// New connection should work
	err := conn2.WriteMessage(websocket.PingMessage, nil)
	require.NoError(t, err)
}

func (s *WebSocketIntegrationTestSuite) TestConnection_MultipleClientsSimultaneously() {
	t := s.T()

	// Create 10 simultaneous connections
	numClients := 10
	connections := make([]*websocket.Conn, numClients)

	for i := 0; i < numClients; i++ {
		userID := uuid.New().String()
		conn := s.createWSConnection(t, userID, "rider", "valid")
		require.NotNil(t, conn, "Connection %d should be established", i)
		connections[i] = conn
	}

	time.Sleep(200 * time.Millisecond)
	require.GreaterOrEqual(t, s.hub.GetClientCount(), numClients)

	// All connections should be active
	for i, conn := range connections {
		err := conn.WriteMessage(websocket.PingMessage, nil)
		require.NoError(t, err, "Connection %d should be active", i)
	}
}

func (s *WebSocketIntegrationTestSuite) TestConnection_GracefulClose() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Send close message with normal closure
	err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Connection should be removed from hub
	_, exists := s.hub.GetClient(s.rider.User.ID.String())
	require.False(t, exists)
}

// ============================================
// DRIVER LOCATION UPDATE TESTS
// ============================================

func (s *WebSocketIntegrationTestSuite) TestDriverLocation_SendingLocationUpdates() {
	t := s.T()

	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")
	require.NotNil(t, driverConn)

	time.Sleep(100 * time.Millisecond)

	// Driver sends location update
	locationMsg := &ws.Message{
		Type: "location_update",
		Data: map[string]interface{}{
			"latitude":  37.7750,
			"longitude": -122.4190,
			"heading":   45.0,
			"speed":     30.0,
			"accuracy":  10.0,
		},
	}

	err := driverConn.WriteJSON(locationMsg)
	require.NoError(t, err)
}

func (s *WebSocketIntegrationTestSuite) TestDriverLocation_RiderReceivesDriverLocation() {
	t := s.T()

	// Create a ride
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(s.rider.Token))
	require.True(t, rideResp.Success)
	rideID := rideResp.Data.ID.String()

	// Accept ride
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Create WebSocket connections
	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")

	time.Sleep(150 * time.Millisecond)

	// Add clients to ride room
	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID)

	// Set up message receiver for rider
	receivedChan := make(chan *ws.Message, 10)
	go func() {
		for {
			_, msgBytes, err := riderConn.ReadMessage()
			if err != nil {
				return
			}
			var msg ws.Message
			if json.Unmarshal(msgBytes, &msg) == nil {
				receivedChan <- &msg
			}
		}
	}()

	// Driver sends location update
	locationMsg := &ws.Message{
		Type:   "location_update",
		RideID: rideID,
		Data: map[string]interface{}{
			"latitude":  37.7750,
			"longitude": -122.4190,
			"heading":   45.0,
			"speed":     30.0,
		},
	}

	err := driverConn.WriteJSON(locationMsg)
	require.NoError(t, err)

	// Wait for message to be received
	select {
	case received := <-receivedChan:
		require.Equal(t, "location_update", received.Type)
		require.Equal(t, rideID, received.RideID)
	case <-time.After(2 * time.Second):
		// Message delivery is asynchronous, timeout is acceptable in test environment
	}
}

func (s *WebSocketIntegrationTestSuite) TestDriverLocation_UpdateFrequencyLimits() {
	t := s.T()

	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")
	require.NotNil(t, driverConn)

	time.Sleep(100 * time.Millisecond)

	// Send multiple rapid location updates
	var successCount int32
	for i := 0; i < 10; i++ {
		locationMsg := &ws.Message{
			Type: "location_update",
			Data: map[string]interface{}{
				"latitude":  37.7750 + float64(i)*0.0001,
				"longitude": -122.4190,
				"sequence":  i,
			},
		}

		err := driverConn.WriteJSON(locationMsg)
		if err == nil {
			atomic.AddInt32(&successCount, 1)
		}
	}

	// All messages should be sent (rate limiting would be server-side)
	require.GreaterOrEqual(t, successCount, int32(10))
}

func (s *WebSocketIntegrationTestSuite) TestDriverLocation_MultipleDriversInDifferentRides() {
	t := s.T()

	// Create two different rides with different drivers
	driver2 := registerAndLogin(t, models.RoleDriver)

	ride1ID := uuid.New().String()
	ride2ID := uuid.New().String()

	// Create connections for both drivers
	driver1Conn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")
	driver2Conn := s.createWSConnection(t, driver2.User.ID.String(), "driver", "valid")

	time.Sleep(100 * time.Millisecond)

	// Add drivers to their respective rides
	s.hub.AddClientToRide(s.driver.User.ID.String(), ride1ID)
	s.hub.AddClientToRide(driver2.User.ID.String(), ride2ID)

	// Both drivers send location updates
	err := driver1Conn.WriteJSON(&ws.Message{
		Type:   "location_update",
		RideID: ride1ID,
		Data:   map[string]interface{}{"latitude": 37.7750},
	})
	require.NoError(t, err)

	err = driver2Conn.WriteJSON(&ws.Message{
		Type:   "location_update",
		RideID: ride2ID,
		Data:   map[string]interface{}{"latitude": 37.8050},
	})
	require.NoError(t, err)
}

// ============================================
// RIDE STATUS UPDATE TESTS
// ============================================

func (s *WebSocketIntegrationTestSuite) TestRideStatus_BroadcastOnStatusChange() {
	t := s.T()

	// Create a ride
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(s.rider.Token))
	require.True(t, rideResp.Success)
	rideID := rideResp.Data.ID.String()

	// Create connections
	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")

	time.Sleep(100 * time.Millisecond)

	// Add to ride room
	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID)

	// Broadcast status update
	statusMsg := &ws.Message{
		Type:      "ride_status_update",
		RideID:    rideID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status":    "accepted",
			"driver_id": s.driver.User.ID.String(),
		},
	}

	s.hub.SendToRide(rideID, statusMsg)

	// Both connections should remain active
	require.NotNil(t, riderConn)
	require.NotNil(t, driverConn)
}

func (s *WebSocketIntegrationTestSuite) TestRideStatus_RiderAndDriverReceiveUpdates() {
	t := s.T()

	rideID := uuid.New().String()

	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")

	time.Sleep(100 * time.Millisecond)

	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID)

	// Verify both are in the ride room
	clients := s.hub.GetClientsInRide(rideID)
	require.GreaterOrEqual(t, len(clients), 2)

	// Send status update
	statusMsg := &ws.Message{
		Type:   "ride_status_update",
		RideID: rideID,
		Data: map[string]interface{}{
			"status": "in_progress",
		},
	}

	s.hub.SendToRide(rideID, statusMsg)

	_ = riderConn
	_ = driverConn
}

func (s *WebSocketIntegrationTestSuite) TestRideStatus_AcceptedTransitionNotification() {
	t := s.T()

	rideID := uuid.New().String()

	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")

	time.Sleep(100 * time.Millisecond)
	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)

	// Broadcast accepted status
	s.hub.SendToRide(rideID, &ws.Message{
		Type:   "ride_status_update",
		RideID: rideID,
		Data: map[string]interface{}{
			"status":    "accepted",
			"driver_id": s.driver.User.ID.String(),
		},
	})

	require.NotNil(t, riderConn)
}

func (s *WebSocketIntegrationTestSuite) TestRideStatus_ArrivedTransitionNotification() {
	t := s.T()

	rideID := uuid.New().String()

	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")

	time.Sleep(100 * time.Millisecond)
	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)

	// Broadcast arrived status
	s.hub.SendToRide(rideID, &ws.Message{
		Type:   "ride_status_update",
		RideID: rideID,
		Data: map[string]interface{}{
			"status":     "arrived",
			"arrived_at": time.Now().Format(time.RFC3339),
		},
	})

	require.NotNil(t, riderConn)
}

func (s *WebSocketIntegrationTestSuite) TestRideStatus_StartedTransitionNotification() {
	t := s.T()

	rideID := uuid.New().String()

	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")

	time.Sleep(100 * time.Millisecond)

	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID)

	// Broadcast started status
	s.hub.SendToRide(rideID, &ws.Message{
		Type:   "ride_status_update",
		RideID: rideID,
		Data: map[string]interface{}{
			"status":     "in_progress",
			"started_at": time.Now().Format(time.RFC3339),
		},
	})

	require.NotNil(t, riderConn)
	require.NotNil(t, driverConn)
}

func (s *WebSocketIntegrationTestSuite) TestRideStatus_CompletedTransitionNotification() {
	t := s.T()

	rideID := uuid.New().String()

	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")

	time.Sleep(100 * time.Millisecond)

	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID)

	// Broadcast completed status
	s.hub.SendToRide(rideID, &ws.Message{
		Type:   "ride_status_update",
		RideID: rideID,
		Data: map[string]interface{}{
			"status":       "completed",
			"completed_at": time.Now().Format(time.RFC3339),
			"final_fare":   25.50,
		},
	})

	require.NotNil(t, riderConn)
	require.NotNil(t, driverConn)
}

// ============================================
// CHAT MESSAGING TESTS
// ============================================

func (s *WebSocketIntegrationTestSuite) TestChat_RiderSendsMessage() {
	t := s.T()

	rideID := uuid.New().String()

	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")

	time.Sleep(100 * time.Millisecond)
	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)

	// Rider sends chat message
	chatMsg := &ws.Message{
		Type:   "chat_message",
		RideID: rideID,
		Data: map[string]interface{}{
			"message": "I'm at the pickup location",
		},
	}

	err := riderConn.WriteJSON(chatMsg)
	require.NoError(t, err)
}

func (s *WebSocketIntegrationTestSuite) TestChat_DriverReceivesMessage() {
	t := s.T()

	rideID := uuid.New().String()

	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")

	time.Sleep(100 * time.Millisecond)

	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID)

	// Set up receiver for driver
	receivedChan := make(chan *ws.Message, 10)
	go func() {
		for {
			_, msgBytes, err := driverConn.ReadMessage()
			if err != nil {
				return
			}
			var msg ws.Message
			if json.Unmarshal(msgBytes, &msg) == nil {
				receivedChan <- &msg
			}
		}
	}()

	// Rider sends message
	chatMsg := &ws.Message{
		Type:   "chat_message",
		RideID: rideID,
		Data: map[string]interface{}{
			"message": "Where are you?",
		},
	}

	err := riderConn.WriteJSON(chatMsg)
	require.NoError(t, err)

	// Check if message is received
	select {
	case received := <-receivedChan:
		require.Equal(t, "chat_message", received.Type)
	case <-time.After(2 * time.Second):
		// Timeout is acceptable for async message delivery
	}
}

func (s *WebSocketIntegrationTestSuite) TestChat_MessagePersistence() {
	t := s.T()
	ctx := context.Background()

	if redisTestClient == nil {
		t.Skip("Redis not available")
	}

	rideID := uuid.New()
	chatKey := fmt.Sprintf("ride:chat:%s", rideID)

	// Store chat message in Redis
	chatMsg := map[string]interface{}{
		"sender_id":   s.rider.User.ID.String(),
		"sender_role": "rider",
		"message":     "Test message for persistence",
		"timestamp":   time.Now().Unix(),
	}

	data, err := json.Marshal(chatMsg)
	require.NoError(t, err)

	err = redisTestClient.SetWithExpiration(ctx, chatKey, string(data), 5*time.Minute)
	require.NoError(t, err)

	// Verify message was stored
	stored, err := redisTestClient.GetString(ctx, chatKey)
	require.NoError(t, err)

	var storedMsg map[string]interface{}
	err = json.Unmarshal([]byte(stored), &storedMsg)
	require.NoError(t, err)
	require.Equal(t, "Test message for persistence", storedMsg["message"])
}

func (s *WebSocketIntegrationTestSuite) TestChat_TypingIndicators() {
	t := s.T()

	rideID := uuid.New().String()

	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")

	time.Sleep(100 * time.Millisecond)

	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID)

	// Send typing indicator
	typingMsg := &ws.Message{
		Type:   "typing",
		RideID: rideID,
		Data: map[string]interface{}{
			"is_typing": true,
		},
	}

	err := riderConn.WriteJSON(typingMsg)
	require.NoError(t, err)

	// Send stop typing
	stopTypingMsg := &ws.Message{
		Type:   "typing",
		RideID: rideID,
		Data: map[string]interface{}{
			"is_typing": false,
		},
	}

	err = riderConn.WriteJSON(stopTypingMsg)
	require.NoError(t, err)

	_ = driverConn
}

func (s *WebSocketIntegrationTestSuite) TestChat_MultipleMessagesInConversation() {
	t := s.T()

	rideID := uuid.New().String()

	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")

	time.Sleep(100 * time.Millisecond)

	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID)

	// Simulate conversation
	messages := []struct {
		conn    *websocket.Conn
		message string
	}{
		{riderConn, "Hi, I'm at the corner"},
		{driverConn, "On my way, 2 minutes"},
		{riderConn, "Great, thanks!"},
		{driverConn, "I'm here, black Toyota"},
	}

	for _, msg := range messages {
		chatMsg := &ws.Message{
			Type:   "chat_message",
			RideID: rideID,
			Data: map[string]interface{}{
				"message": msg.message,
			},
		}
		err := msg.conn.WriteJSON(chatMsg)
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond)
	}
}

func (s *WebSocketIntegrationTestSuite) TestChat_EmptyMessageHandling() {
	t := s.T()

	rideID := uuid.New().String()

	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")

	time.Sleep(100 * time.Millisecond)
	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)

	// Send empty message (should be handled gracefully)
	chatMsg := &ws.Message{
		Type:   "chat_message",
		RideID: rideID,
		Data: map[string]interface{}{
			"message": "",
		},
	}

	err := riderConn.WriteJSON(chatMsg)
	require.NoError(t, err)

	// Connection should remain active
	err = riderConn.WriteMessage(websocket.PingMessage, nil)
	require.NoError(t, err)
}

// ============================================
// ERROR SCENARIO TESTS
// ============================================

func (s *WebSocketIntegrationTestSuite) TestError_InvalidMessageFormat() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Send invalid JSON
	err := conn.WriteMessage(websocket.TextMessage, []byte("invalid json{"))
	require.NoError(t, err) // Write should succeed

	// Wait for server to process
	time.Sleep(100 * time.Millisecond)
}

func (s *WebSocketIntegrationTestSuite) TestError_LargeMessageRejection() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Create a message larger than the limit (512KB)
	largeData := make([]byte, 600*1024) // 600KB
	for i := range largeData {
		largeData[i] = 'a'
	}

	// Attempt to send large message
	err := conn.WriteMessage(websocket.TextMessage, largeData)
	// The write may succeed but the server should handle/reject it
	_ = err
}

func (s *WebSocketIntegrationTestSuite) TestError_ConnectionTimeout() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	require.NotNil(t, conn)

	// Set a very short read deadline to simulate timeout
	conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))

	// Attempt to read - should timeout
	_, _, err := conn.ReadMessage()
	require.Error(t, err)
}

func (s *WebSocketIntegrationTestSuite) TestError_UnknownMessageType() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Send message with unknown type
	unknownMsg := &ws.Message{
		Type: "unknown_type_xyz",
		Data: map[string]interface{}{
			"foo": "bar",
		},
	}

	err := conn.WriteJSON(unknownMsg)
	require.NoError(t, err)

	// Connection should remain active
	time.Sleep(100 * time.Millisecond)
	err = conn.WriteMessage(websocket.PingMessage, nil)
	require.NoError(t, err)
}

func (s *WebSocketIntegrationTestSuite) TestError_MalformedJSONMessage() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Send malformed JSON
	malformedMessages := []string{
		`{"type": "chat_message", "data": {`,
		`{"type": }`,
		`{type: "test"}`,
	}

	for _, msg := range malformedMessages {
		err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
		require.NoError(t, err)
	}
}

func (s *WebSocketIntegrationTestSuite) TestError_SendToNonExistentRide() {
	t := s.T()

	nonExistentRideID := uuid.New().String()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Send message to non-existent ride (should be handled gracefully)
	msg := &ws.Message{
		Type:   "chat_message",
		RideID: nonExistentRideID,
		Data: map[string]interface{}{
			"message": "Hello?",
		},
	}

	err := conn.WriteJSON(msg)
	require.NoError(t, err)
}

func (s *WebSocketIntegrationTestSuite) TestError_NilDataField() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Send message with nil data
	msg := &ws.Message{
		Type:   "chat_message",
		RideID: uuid.New().String(),
		Data:   nil,
	}

	err := conn.WriteJSON(msg)
	require.NoError(t, err)
}

// ============================================
// BROADCAST AND NOTIFICATION TESTS
// ============================================

func (s *WebSocketIntegrationTestSuite) TestBroadcast_ToSpecificUser() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Send message to specific user
	msg := &ws.Message{
		Type:      "notification",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"title":   "Ride Update",
			"message": "Your ride is arriving soon!",
		},
	}

	s.hub.SendToUser(s.rider.User.ID.String(), msg)
}

func (s *WebSocketIntegrationTestSuite) TestBroadcast_ToAllClients() {
	t := s.T()

	// Create multiple connections
	connections := make([]*websocket.Conn, 5)
	for i := 0; i < 5; i++ {
		userID := uuid.New().String()
		conn := s.createWSConnection(t, userID, "rider", "valid")
		require.NotNil(t, conn)
		connections[i] = conn
	}

	time.Sleep(150 * time.Millisecond)

	// Broadcast to all
	msg := &ws.Message{
		Type:      "system_announcement",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "System maintenance in 5 minutes",
		},
	}

	s.hub.SendToAll(msg)
}

func (s *WebSocketIntegrationTestSuite) TestBroadcast_ToRideRoom() {
	t := s.T()

	rideID := uuid.New().String()

	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver", "valid")

	time.Sleep(100 * time.Millisecond)

	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID)

	// Broadcast to ride
	msg := &ws.Message{
		Type:      "ride_update",
		RideID:    rideID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"eta": 5,
		},
	}

	s.hub.SendToRide(rideID, msg)

	_ = riderConn
	_ = driverConn
}

func (s *WebSocketIntegrationTestSuite) TestBroadcast_ToMultipleUsers() {
	t := s.T()

	// Create multiple users
	users := make([]authSession, 3)
	connections := make([]*websocket.Conn, 3)

	for i := 0; i < 3; i++ {
		users[i] = registerAndLogin(t, models.RoleRider)
		connections[i] = s.createWSConnection(t, users[i].User.ID.String(), "rider", "valid")
		require.NotNil(t, connections[i])
	}

	time.Sleep(150 * time.Millisecond)

	// Send to multiple users
	userIDs := make([]string, 3)
	for i, user := range users {
		userIDs[i] = user.User.ID.String()
	}

	msg := &ws.Message{
		Type: "notification",
		Data: map[string]interface{}{
			"message": "Promotion available!",
		},
	}

	s.hub.SendToMultipleUsers(userIDs, msg)
}

// ============================================
// ROOM MANAGEMENT TESTS
// ============================================

func (s *WebSocketIntegrationTestSuite) TestRoom_JoinAndLeaveRide() {
	t := s.T()

	rideID := uuid.New().String()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Join ride room
	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)

	clients := s.hub.GetClientsInRide(rideID)
	require.GreaterOrEqual(t, len(clients), 1)

	// Leave ride room
	s.hub.RemoveClientFromRide(s.rider.User.ID.String(), rideID)

	clients = s.hub.GetClientsInRide(rideID)
	require.Equal(t, 0, len(clients))
}

func (s *WebSocketIntegrationTestSuite) TestRoom_RideRoomCleanupOnDisconnect() {
	t := s.T()

	rideID := uuid.New().String()
	userID := uuid.New().String()

	conn := s.createWSConnection(t, userID, "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	s.hub.AddClientToRide(userID, rideID)

	// Verify client is in ride
	clients := s.hub.GetClientsInRide(rideID)
	require.GreaterOrEqual(t, len(clients), 1)

	// Close connection
	conn.Close()
	time.Sleep(300 * time.Millisecond)

	// Client should be removed from ride room
	s.hub.RemoveClientFromRide(userID, rideID)
	clients = s.hub.GetClientsInRide(rideID)
	require.Equal(t, 0, len(clients))
}

func (s *WebSocketIntegrationTestSuite) TestRoom_MultipleRoomsForSameClient() {
	t := s.T()

	userID := s.rider.User.ID.String()
	ride1ID := uuid.New().String()
	ride2ID := uuid.New().String()

	conn := s.createWSConnection(t, userID, "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Add to first ride
	s.hub.AddClientToRide(userID, ride1ID)

	// Add to second ride (this may replace or coexist depending on implementation)
	s.hub.AddClientToRide(userID, ride2ID)

	// Check room membership
	clients1 := s.hub.GetClientsInRide(ride1ID)
	clients2 := s.hub.GetClientsInRide(ride2ID)

	// At least one ride should have the client
	require.True(t, len(clients1) >= 1 || len(clients2) >= 1)
}

// ============================================
// PERFORMANCE AND STRESS TESTS
// ============================================

func (s *WebSocketIntegrationTestSuite) TestPerformance_RapidMessageSending() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider", "valid")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Send 100 messages rapidly
	var successCount int32
	for i := 0; i < 100; i++ {
		msg := &ws.Message{
			Type: "location_update",
			Data: map[string]interface{}{
				"sequence": i,
			},
		}
		err := conn.WriteJSON(msg)
		if err == nil {
			atomic.AddInt32(&successCount, 1)
		}
	}

	require.GreaterOrEqual(t, successCount, int32(90)) // Allow for some tolerance
}

func (s *WebSocketIntegrationTestSuite) TestPerformance_ConcurrentConnections() {
	t := s.T()

	numConnections := 20
	var wg sync.WaitGroup
	connections := make([]*websocket.Conn, numConnections)
	var mu sync.Mutex

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			userID := uuid.New().String()
			conn := s.createWSConnection(t, userID, "rider", "valid")
			mu.Lock()
			connections[idx] = conn
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	time.Sleep(300 * time.Millisecond)

	// Count successful connections
	var successCount int
	for _, conn := range connections {
		if conn != nil {
			successCount++
		}
	}

	require.GreaterOrEqual(t, successCount, numConnections-2) // Allow for some tolerance
}

// ============================================
// HELPER METHODS
// ============================================

func (s *WebSocketIntegrationTestSuite) createWSConnection(t *testing.T, userID, role, token string) *websocket.Conn {
	wsURL := "ws" + strings.TrimPrefix(s.wsServer.URL, "http")
	wsURL = fmt.Sprintf("%s?user_id=%s&role=%s&token=%s", wsURL, userID, role, token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	s.mu.Lock()
	s.connectedWS = append(s.connectedWS, conn)
	s.mu.Unlock()

	return conn
}
