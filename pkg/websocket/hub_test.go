package websocket

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestNewHub tests hub creation
func TestNewHub(t *testing.T) {
	hub := NewHub()

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.rides)
	assert.NotNil(t, hub.Register)
	assert.NotNil(t, hub.Unregister)
	assert.NotNil(t, hub.Broadcast)
	assert.NotNil(t, hub.handlers)
}

// TestRegisterClient tests client registration
func TestRegisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Verify client is registered
	registeredClient, ok := hub.GetClient("user-123")
	assert.True(t, ok)
	assert.Equal(t, client.ID, registeredClient.ID)
	assert.Equal(t, 1, hub.GetClientCount())
}

// TestRegisterDuplicateClient tests replacing existing client
func TestRegisterDuplicateClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Register first client
	conn1 := createTestWebSocketConn(t)
	client1 := NewClient("user-123", conn1, hub, "rider", zap.NewNop())

	hub.Register <- client1
	time.Sleep(10 * time.Millisecond)

	// Register second client with same ID
	conn2 := createTestWebSocketConn(t)
	client2 := NewClient("user-123", conn2, hub, "rider", zap.NewNop())

	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	// Verify only one client exists
	assert.Equal(t, 1, hub.GetClientCount())

	registeredClient, ok := hub.GetClient("user-123")
	assert.True(t, ok)
	assert.Equal(t, client2.ID, registeredClient.ID)
}

// TestUnregisterClient tests client unregistration
func TestUnregisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	// Register client
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.GetClientCount())

	// Unregister client
	hub.Unregister <- client
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 0, hub.GetClientCount())

	_, ok := hub.GetClient("user-123")
	assert.False(t, ok)
}

// TestUnregisterClientFromRide tests removing client from ride on unregister
func TestUnregisterClientFromRide(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	// Register client and add to ride
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	rideID := "ride-789"
	hub.AddClientToRide(client.ID, rideID)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.GetRideCount())
	assert.Len(t, hub.GetClientsInRide(rideID), 1)

	// Unregister client
	hub.Unregister <- client
	time.Sleep(10 * time.Millisecond)

	// Verify client is removed from ride
	assert.Equal(t, 0, hub.GetRideCount())
	assert.Len(t, hub.GetClientsInRide(rideID), 0)
}

// TestAddClientToRide tests adding client to ride room
func TestAddClientToRide(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	// Register client
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	rideID := "ride-789"

	// Add to ride
	hub.AddClientToRide(client.ID, rideID)
	time.Sleep(10 * time.Millisecond)

	// Verify
	assert.Equal(t, 1, hub.GetRideCount())
	assert.Len(t, hub.GetClientsInRide(rideID), 1)
	assert.Equal(t, rideID, client.GetRide())
}

// TestAddMultipleClientsToRide tests adding multiple clients to same ride
func TestAddMultipleClientsToRide(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create two clients
	conn1 := createTestWebSocketConn(t)
	client1 := NewClient("rider-123", conn1, hub, "rider", zap.NewNop())

	conn2 := createTestWebSocketConn(t)
	client2 := NewClient("driver-456", conn2, hub, "driver", zap.NewNop())

	// Register clients
	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	rideID := "ride-789"

	// Add both to same ride
	hub.AddClientToRide(client1.ID, rideID)
	hub.AddClientToRide(client2.ID, rideID)
	time.Sleep(10 * time.Millisecond)

	// Verify
	assert.Equal(t, 1, hub.GetRideCount())
	assert.Len(t, hub.GetClientsInRide(rideID), 2)
}

// TestRemoveClientFromRide tests removing client from ride room
func TestRemoveClientFromRide(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	// Register client and add to ride
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	rideID := "ride-789"
	hub.AddClientToRide(client.ID, rideID)
	time.Sleep(10 * time.Millisecond)

	// Remove from ride
	hub.RemoveClientFromRide(client.ID, rideID)
	time.Sleep(10 * time.Millisecond)

	// Verify
	assert.Equal(t, 0, hub.GetRideCount())
	assert.Len(t, hub.GetClientsInRide(rideID), 0)
	assert.Equal(t, "", client.GetRide())
}

// TestRemoveLastClientFromRide tests ride room cleanup
func TestRemoveLastClientFromRide(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create two clients
	conn1 := createTestWebSocketConn(t)
	client1 := NewClient("rider-123", conn1, hub, "rider", zap.NewNop())

	conn2 := createTestWebSocketConn(t)
	client2 := NewClient("driver-456", conn2, hub, "driver", zap.NewNop())

	// Register and add to ride
	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	rideID := "ride-789"
	hub.AddClientToRide(client1.ID, rideID)
	hub.AddClientToRide(client2.ID, rideID)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.GetRideCount())

	// Remove first client
	hub.RemoveClientFromRide(client1.ID, rideID)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.GetRideCount()) // Ride still exists
	assert.Len(t, hub.GetClientsInRide(rideID), 1)

	// Remove second client
	hub.RemoveClientFromRide(client2.ID, rideID)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 0, hub.GetRideCount()) // Ride removed
	assert.Len(t, hub.GetClientsInRide(rideID), 0)
}

// TestSendToUser tests sending message to specific user
func TestSendToUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Send message
	msg := &Message{
		Type: "test",
		Data: map[string]interface{}{
			"message": "Hello",
		},
	}

	hub.SendToUser(client.ID, msg)
	time.Sleep(10 * time.Millisecond)

	// Message should be in client's send channel
	select {
	case receivedMsg := <-client.Send:
		assert.Equal(t, msg.Type, receivedMsg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received")
	}
}

// TestSendToNonExistentUser tests sending to non-existent user
func TestSendToNonExistentUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Send message to non-existent user
	msg := &Message{
		Type: "test",
		Data: map[string]interface{}{
			"message": "Hello",
		},
	}

	// Should not panic
	hub.SendToUser("non-existent", msg)
	time.Sleep(10 * time.Millisecond)
}

// TestSendToRide tests sending message to all clients in ride
func TestSendToRide(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create two clients
	conn1 := createTestWebSocketConn(t)
	client1 := NewClient("rider-123", conn1, hub, "rider", zap.NewNop())

	conn2 := createTestWebSocketConn(t)
	client2 := NewClient("driver-456", conn2, hub, "driver", zap.NewNop())

	// Register and add to ride
	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	rideID := "ride-789"
	hub.AddClientToRide(client1.ID, rideID)
	hub.AddClientToRide(client2.ID, rideID)
	time.Sleep(10 * time.Millisecond)

	// Send message to ride
	msg := &Message{
		Type:   "ride_update",
		RideID: rideID,
		Data: map[string]interface{}{
			"status": "in_progress",
		},
	}

	hub.SendToRide(rideID, msg)
	time.Sleep(10 * time.Millisecond)

	// Both clients should receive the message
	select {
	case <-client1.Send:
		// Message received
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client 1 did not receive message")
	}

	select {
	case <-client2.Send:
		// Message received
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client 2 did not receive message")
	}
}

// TestSendToAll tests broadcasting to all clients
func TestSendToAll(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create three clients
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		conn := createTestWebSocketConn(t)
		client := NewClient("user-"+string(rune(i)), conn, hub, "rider", zap.NewNop())
		clients[i] = client
		hub.Register <- client
	}

	time.Sleep(10 * time.Millisecond)

	// Send to all
	msg := &Message{
		Type: "announcement",
		Data: map[string]interface{}{
			"message": "System maintenance in 5 minutes",
		},
	}

	hub.SendToAll(msg)
	time.Sleep(10 * time.Millisecond)

	// All clients should receive
	for i, client := range clients {
		select {
		case <-client.Send:
			// Message received
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Client %d did not receive broadcast", i)
		}
	}
}

// TestRegisterHandler tests handler registration
func TestRegisterHandler(t *testing.T) {
	hub := NewHub()

	handlerCalled := false
	handler := func(client *Client, msg *Message) {
		handlerCalled = true
	}

	hub.RegisterHandler("test_message", handler)

	// Verify handler is registered
	assert.Contains(t, hub.handlers, "test_message")

	// Test handler is called
	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	msg := &Message{
		Type: "test_message",
		Data: map[string]interface{}{},
	}

	hub.HandleMessage(client, msg)

	assert.True(t, handlerCalled)
}

// TestHandleMessageUnknownType tests handling unknown message type
func TestHandleMessageUnknownType(t *testing.T) {
	hub := NewHub()

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	msg := &Message{
		Type: "unknown_type",
		Data: map[string]interface{}{},
	}

	// Should not panic
	hub.HandleMessage(client, msg)
}

// TestConcurrentAccess tests thread-safety under concurrent load
func TestConcurrentAccess(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	var wg sync.WaitGroup
	numClients := 50

	// Register many clients concurrently
	wg.Add(numClients)
	for i := 0; i < numClients; i++ {
		go func(id int) {
			defer wg.Done()

			conn := createTestWebSocketConn(t)
			client := NewClient("user-"+string(rune(id)), conn, hub, "rider", zap.NewNop())

			hub.Register <- client
			time.Sleep(1 * time.Millisecond)

			// Add to random ride
			rideID := "ride-" + string(rune(id%10))
			hub.AddClientToRide(client.ID, rideID)

			// Send some messages
			for j := 0; j < 5; j++ {
				msg := &Message{
					Type: "test",
					Data: map[string]interface{}{
						"count": j,
					},
				}
				hub.SendToUser(client.ID, msg)
			}

			// Unregister
			hub.Unregister <- client
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// All clients should be unregistered
	assert.Equal(t, 0, hub.GetClientCount())
	assert.Equal(t, 0, hub.GetRideCount())
}

// TestGetClient tests retrieving specific client
func TestGetClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Get existing client
	retrievedClient, ok := hub.GetClient("user-123")
	assert.True(t, ok)
	assert.Equal(t, client.ID, retrievedClient.ID)

	// Get non-existent client
	_, ok = hub.GetClient("non-existent")
	assert.False(t, ok)
}

// TestGetClientsInRide tests retrieving all clients in a ride
func TestGetClientsInRide(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create three clients
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		conn := createTestWebSocketConn(t)
		client := NewClient("user-"+string(rune(i)), conn, hub, "rider", zap.NewNop())
		clients[i] = client
		hub.Register <- client
	}

	time.Sleep(10 * time.Millisecond)

	rideID := "ride-789"

	// Add first two clients to ride
	hub.AddClientToRide(clients[0].ID, rideID)
	hub.AddClientToRide(clients[1].ID, rideID)
	time.Sleep(10 * time.Millisecond)

	// Get clients in ride
	rideClients := hub.GetClientsInRide(rideID)
	assert.Len(t, rideClients, 2)

	// Get clients in non-existent ride
	noClients := hub.GetClientsInRide("non-existent")
	assert.Len(t, noClients, 0)
}

// TestGetClientCount tests counting connected clients
func TestGetClientCount(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	assert.Equal(t, 0, hub.GetClientCount())

	// Register clients
	for i := 0; i < 5; i++ {
		conn := createTestWebSocketConn(t)
		client := NewClient("user-"+string(rune(i)), conn, hub, "rider", zap.NewNop())
		hub.Register <- client
	}

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 5, hub.GetClientCount())
}

// TestGetRideCount tests counting active rides
func TestGetRideCount(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	assert.Equal(t, 0, hub.GetRideCount())

	// Create clients
	clients := make([]*Client, 6)
	for i := 0; i < 6; i++ {
		conn := createTestWebSocketConn(t)
		client := NewClient("user-"+string(rune(i)), conn, hub, "rider", zap.NewNop())
		clients[i] = client
		hub.Register <- client
	}

	time.Sleep(10 * time.Millisecond)

	// Create 3 rides with 2 clients each
	for i := 0; i < 3; i++ {
		rideID := "ride-" + string(rune(i))
		hub.AddClientToRide(clients[i*2].ID, rideID)
		hub.AddClientToRide(clients[i*2+1].ID, rideID)
	}

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 3, hub.GetRideCount())
}

// TestBroadcastChannelCapacity tests broadcast channel buffering
func TestBroadcastChannelCapacity(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Send many messages rapidly
	for i := 0; i < 300; i++ {
		msg := &Message{
			Type: "test",
			Data: map[string]interface{}{
				"count": i,
			},
		}
		hub.SendToUser(client.ID, msg)
	}

	// Should not deadlock or panic
	time.Sleep(50 * time.Millisecond)
}

// TestMessageRouting tests complete message flow
func TestMessageRouting(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	messageReceived := false
	var receivedMessage *Message

	// Register handler
	hub.RegisterHandler("custom_type", func(c *Client, msg *Message) {
		messageReceived = true
		receivedMessage = msg
	})

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Send message
	msg := &Message{
		Type: "custom_type",
		Data: map[string]interface{}{
			"test_data": "test_value",
		},
	}

	hub.HandleMessage(client, msg)

	// Verify handler was called
	assert.True(t, messageReceived)
	assert.Equal(t, msg.Type, receivedMessage.Type)
	assert.Equal(t, "test_value", receivedMessage.Data["test_data"])
}

// TestClientChannelOverflow tests handling of slow/stuck clients
func TestClientChannelOverflow(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	// Use small channel for testing
	client.Send = make(chan *Message, 2)

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Fill the channel beyond capacity
	for i := 0; i < 5; i++ {
		msg := &Message{
			Type: "test",
			Data: map[string]interface{}{
				"count": i,
			},
		}
		client.SendMessage(msg)
	}

	time.Sleep(10 * time.Millisecond)

	// Client should handle overflow gracefully (channel closed)
}
