package chat

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ===== Mock Implementations =====

// MockChatRepository is a mock implementation of RepositoryInterface
type MockChatRepository struct {
	mock.Mock
}

func (m *MockChatRepository) SaveMessage(ctx context.Context, msg *ChatMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockChatRepository) GetMessagesByRide(ctx context.Context, rideID uuid.UUID, limit, offset int) ([]ChatMessage, error) {
	args := m.Called(ctx, rideID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ChatMessage), args.Error(1)
}

func (m *MockChatRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*ChatMessage, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChatMessage), args.Error(1)
}

func (m *MockChatRepository) MarkMessagesDelivered(ctx context.Context, rideID, recipientID uuid.UUID) error {
	args := m.Called(ctx, rideID, recipientID)
	return args.Error(0)
}

func (m *MockChatRepository) MarkMessagesRead(ctx context.Context, rideID, recipientID, lastReadID uuid.UUID) error {
	args := m.Called(ctx, rideID, recipientID, lastReadID)
	return args.Error(0)
}

func (m *MockChatRepository) GetUnreadCount(ctx context.Context, rideID, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, rideID, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockChatRepository) GetLastMessage(ctx context.Context, rideID uuid.UUID) (*ChatMessage, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChatMessage), args.Error(1)
}

func (m *MockChatRepository) DeleteMessagesByRide(ctx context.Context, rideID uuid.UUID) error {
	args := m.Called(ctx, rideID)
	return args.Error(0)
}

func (m *MockChatRepository) GetMessageCount(ctx context.Context, rideID uuid.UUID) (int, error) {
	args := m.Called(ctx, rideID)
	return args.Int(0), args.Error(1)
}

func (m *MockChatRepository) GetQuickReplies(ctx context.Context, role string) ([]QuickReply, error) {
	args := m.Called(ctx, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]QuickReply), args.Error(1)
}

func (m *MockChatRepository) CreateQuickReply(ctx context.Context, qr *QuickReply) error {
	args := m.Called(ctx, qr)
	return args.Error(0)
}

func (m *MockChatRepository) GetActiveConversations(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

// MockChatHub is a mock implementation of HubInterface
type MockChatHub struct {
	mock.Mock
}

func (m *MockChatHub) SendToRide(rideID string, msg *ws.Message) {
	m.Called(rideID, msg)
}

func (m *MockChatHub) SendToUser(userID string, msg *ws.Message) {
	m.Called(userID, msg)
}

// ===== SendMessage Tests =====

func TestService_SendMessage_Success(t *testing.T) {
	tests := []struct {
		name        string
		messageType MessageType
		content     string
		imageURL    *string
		latitude    *float64
		longitude   *float64
	}{
		{
			name:        "text message",
			messageType: MessageTypeText,
			content:     "Hello, where are you?",
		},
		{
			name:        "image message",
			messageType: MessageTypeImage,
			content:     "Check this out",
			imageURL:    stringPtr("https://example.com/image.jpg"),
		},
		{
			name:        "location message",
			messageType: MessageTypeLocation,
			content:     "I am here",
			latitude:    float64Ptr(37.7749),
			longitude:   float64Ptr(-122.4194),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockChatRepository)
			mockHub := new(MockChatHub)
			service := NewService(mockRepo, mockHub)

			ctx := context.Background()
			senderID := uuid.New()
			rideID := uuid.New()

			req := &SendMessageRequest{
				RideID:      rideID,
				MessageType: tt.messageType,
				Content:     tt.content,
				ImageURL:    tt.imageURL,
				Latitude:    tt.latitude,
				Longitude:   tt.longitude,
			}

			mockRepo.On("SaveMessage", ctx, mock.AnythingOfType("*chat.ChatMessage")).Return(nil)
			mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()

			// Act
			msg, err := service.SendMessage(ctx, senderID, "rider", req)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, msg)
			assert.Equal(t, senderID, msg.SenderID)
			assert.Equal(t, rideID, msg.RideID)
			assert.Equal(t, "rider", msg.SenderRole)
			assert.Equal(t, tt.messageType, msg.MessageType)
			assert.Equal(t, tt.content, msg.Content)
			assert.Equal(t, MessageStatusSent, msg.Status)
			mockRepo.AssertExpectations(t)
			mockHub.AssertExpectations(t)
		})
	}
}

func TestService_SendMessage_InvalidMessageType(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	senderID := uuid.New()

	req := &SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: "invalid_type",
		Content:     "Test",
	}

	// Act
	msg, err := service.SendMessage(ctx, senderID, "rider", req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, msg)
	appErr, ok := err.(*common.AppError)
	assert.True(t, ok)
	assert.Contains(t, appErr.Message, "invalid message type")
}

func TestService_SendMessage_EmptyContent(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	senderID := uuid.New()

	req := &SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeText,
		Content:     "",
	}

	// Act
	msg, err := service.SendMessage(ctx, senderID, "rider", req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, msg)
	appErr, ok := err.(*common.AppError)
	assert.True(t, ok)
	assert.Contains(t, appErr.Message, "cannot be empty")
}

func TestService_SendMessage_ContentTooLong(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	senderID := uuid.New()

	// Create content longer than maxTextLength (1000)
	longContent := strings.Repeat("a", 1001)

	req := &SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeText,
		Content:     longContent,
	}

	// Act
	msg, err := service.SendMessage(ctx, senderID, "rider", req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, msg)
	appErr, ok := err.(*common.AppError)
	assert.True(t, ok)
	assert.Contains(t, appErr.Message, "message too long")
}

func TestService_SendMessage_ImageURLTooLong(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	senderID := uuid.New()

	// Create image URL longer than maxImageURLLen (2048)
	longURL := "https://example.com/" + strings.Repeat("a", 2049)

	req := &SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeImage,
		Content:     "Check this",
		ImageURL:    &longURL,
	}

	// Act
	msg, err := service.SendMessage(ctx, senderID, "rider", req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, msg)
	appErr, ok := err.(*common.AppError)
	assert.True(t, ok)
	assert.Contains(t, appErr.Message, "image URL too long")
}

func TestService_SendMessage_LocationWithoutCoordinates(t *testing.T) {
	tests := []struct {
		name      string
		latitude  *float64
		longitude *float64
	}{
		{
			name:      "missing both coordinates",
			latitude:  nil,
			longitude: nil,
		},
		{
			name:      "missing latitude",
			latitude:  nil,
			longitude: float64Ptr(-122.4194),
		},
		{
			name:      "missing longitude",
			latitude:  float64Ptr(37.7749),
			longitude: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockChatRepository)
			mockHub := new(MockChatHub)
			service := NewService(mockRepo, mockHub)

			ctx := context.Background()
			senderID := uuid.New()

			req := &SendMessageRequest{
				RideID:      uuid.New(),
				MessageType: MessageTypeLocation,
				Content:     "I am here",
				Latitude:    tt.latitude,
				Longitude:   tt.longitude,
			}

			// Act
			msg, err := service.SendMessage(ctx, senderID, "rider", req)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, msg)
			appErr, ok := err.(*common.AppError)
			assert.True(t, ok)
			assert.Contains(t, appErr.Message, "require latitude and longitude")
		})
	}
}

func TestService_SendMessage_SaveError(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	senderID := uuid.New()

	req := &SendMessageRequest{
		RideID:      uuid.New(),
		MessageType: MessageTypeText,
		Content:     "Hello",
	}

	mockRepo.On("SaveMessage", ctx, mock.AnythingOfType("*chat.ChatMessage")).
		Return(errors.New("database error"))

	// Act
	msg, err := service.SendMessage(ctx, senderID, "rider", req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "save message")
	mockRepo.AssertExpectations(t)
}

func TestService_SendMessage_NoHub(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil) // No hub

	ctx := context.Background()
	senderID := uuid.New()
	rideID := uuid.New()

	req := &SendMessageRequest{
		RideID:      rideID,
		MessageType: MessageTypeText,
		Content:     "Hello",
	}

	mockRepo.On("SaveMessage", ctx, mock.AnythingOfType("*chat.ChatMessage")).Return(nil)

	// Act
	msg, err := service.SendMessage(ctx, senderID, "driver", req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, "driver", msg.SenderRole)
	mockRepo.AssertExpectations(t)
}

func TestService_SendMessage_WebSocketBroadcast(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	senderID := uuid.New()
	rideID := uuid.New()

	req := &SendMessageRequest{
		RideID:      rideID,
		MessageType: MessageTypeText,
		Content:     "Hello",
	}

	mockRepo.On("SaveMessage", ctx, mock.AnythingOfType("*chat.ChatMessage")).Return(nil)

	// Capture the WebSocket message
	var capturedMsg *ws.Message
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).
		Run(func(args mock.Arguments) {
			capturedMsg = args.Get(1).(*ws.Message)
		}).Return()

	// Act
	_, err := service.SendMessage(ctx, senderID, "rider", req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, capturedMsg)
	assert.Equal(t, "chat.message", capturedMsg.Type)
	assert.Equal(t, rideID.String(), capturedMsg.RideID)
	assert.Equal(t, senderID.String(), capturedMsg.UserID)
	assert.Equal(t, "Hello", capturedMsg.Data["content"])
	assert.Equal(t, "text", capturedMsg.Data["message_type"])
	mockHub.AssertExpectations(t)
}

// ===== SendSystemMessage Tests =====

func TestService_SendSystemMessage_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	rideID := uuid.New()
	content := "Driver has arrived"

	mockRepo.On("SaveMessage", ctx, mock.MatchedBy(func(msg *ChatMessage) bool {
		return msg.RideID == rideID &&
			msg.SenderID == uuid.Nil &&
			msg.SenderRole == "system" &&
			msg.MessageType == MessageTypeSystem &&
			msg.Content == content &&
			msg.Status == MessageStatusSent
	})).Return(nil)

	mockHub.On("SendToRide", rideID.String(), mock.MatchedBy(func(msg *ws.Message) bool {
		return msg.Type == "chat.system" &&
			msg.RideID == rideID.String() &&
			msg.Data["content"] == content
	})).Return()

	// Act
	err := service.SendSystemMessage(ctx, rideID, content)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockHub.AssertExpectations(t)
}

func TestService_SendSystemMessage_SaveError(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	rideID := uuid.New()

	mockRepo.On("SaveMessage", ctx, mock.AnythingOfType("*chat.ChatMessage")).
		Return(errors.New("database error"))

	// Act
	err := service.SendSystemMessage(ctx, rideID, "System message")

	// Assert
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_SendSystemMessage_NoHub(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil)

	ctx := context.Background()
	rideID := uuid.New()

	mockRepo.On("SaveMessage", ctx, mock.AnythingOfType("*chat.ChatMessage")).Return(nil)

	// Act
	err := service.SendSystemMessage(ctx, rideID, "System message")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ===== GetConversation Tests =====

func TestService_GetConversation_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()
	limit := 50
	offset := 0

	now := time.Now()
	messages := []ChatMessage{
		{
			ID:          uuid.New(),
			RideID:      rideID,
			SenderID:    uuid.New(),
			SenderRole:  "driver",
			MessageType: MessageTypeText,
			Content:     "I'm on my way",
			Status:      MessageStatusSent,
			CreatedAt:   now.Add(-5 * time.Minute),
		},
		{
			ID:          uuid.New(),
			RideID:      rideID,
			SenderID:    userID,
			SenderRole:  "rider",
			MessageType: MessageTypeText,
			Content:     "Thanks!",
			Status:      MessageStatusSent,
			CreatedAt:   now,
		},
	}

	lastMsg := &messages[1]

	mockRepo.On("GetMessagesByRide", ctx, rideID, limit, offset).Return(messages, nil)
	mockRepo.On("GetUnreadCount", ctx, rideID, userID).Return(1, nil)
	mockRepo.On("GetLastMessage", ctx, rideID).Return(lastMsg, nil)
	mockRepo.On("MarkMessagesDelivered", ctx, rideID, userID).Return(nil)
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).Return()

	// Act
	conversation, err := service.GetConversation(ctx, userID, rideID, limit, offset)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, conversation)
	assert.Equal(t, rideID, conversation.RideID)
	assert.Len(t, conversation.Messages, 2)
	assert.Equal(t, 1, conversation.UnreadCount)
	assert.Equal(t, lastMsg, conversation.LastMessage)
	mockRepo.AssertExpectations(t)
	mockHub.AssertExpectations(t)
}

func TestService_GetConversation_EmptyMessages(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	mockRepo.On("GetMessagesByRide", ctx, rideID, 50, 0).Return(nil, nil)
	mockRepo.On("GetUnreadCount", ctx, rideID, userID).Return(0, nil)
	mockRepo.On("GetLastMessage", ctx, rideID).Return(nil, pgx.ErrNoRows)
	mockRepo.On("MarkMessagesDelivered", ctx, rideID, userID).Return(nil)

	// Act
	conversation, err := service.GetConversation(ctx, userID, rideID, 50, 0)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, conversation)
	assert.NotNil(t, conversation.Messages)
	assert.Len(t, conversation.Messages, 0)
	mockRepo.AssertExpectations(t)
}

func TestService_GetConversation_GetMessagesError(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	mockRepo.On("GetMessagesByRide", ctx, rideID, 50, 0).Return(nil, errors.New("database error"))

	// Act
	conversation, err := service.GetConversation(ctx, userID, rideID, 50, 0)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, conversation)
	mockRepo.AssertExpectations(t)
}

func TestService_GetConversation_Pagination(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		offset int
	}{
		{name: "first page", limit: 20, offset: 0},
		{name: "second page", limit: 20, offset: 20},
		{name: "large page", limit: 100, offset: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockChatRepository)
			mockHub := new(MockChatHub)
			service := NewService(mockRepo, mockHub)

			ctx := context.Background()
			userID := uuid.New()
			rideID := uuid.New()

			mockRepo.On("GetMessagesByRide", ctx, rideID, tt.limit, tt.offset).Return([]ChatMessage{}, nil)
			mockRepo.On("GetUnreadCount", ctx, rideID, userID).Return(0, nil)
			mockRepo.On("GetLastMessage", ctx, rideID).Return(nil, nil)
			mockRepo.On("MarkMessagesDelivered", ctx, rideID, userID).Return(nil)

			// Act
			_, err := service.GetConversation(ctx, userID, rideID, tt.limit, tt.offset)

			// Assert
			assert.NoError(t, err)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_GetConversation_DeliveredNotification(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	messages := []ChatMessage{{ID: uuid.New()}}

	mockRepo.On("GetMessagesByRide", ctx, rideID, 50, 0).Return(messages, nil)
	mockRepo.On("GetUnreadCount", ctx, rideID, userID).Return(0, nil)
	mockRepo.On("GetLastMessage", ctx, rideID).Return(nil, nil)
	mockRepo.On("MarkMessagesDelivered", ctx, rideID, userID).Return(nil)

	var capturedMsg *ws.Message
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).
		Run(func(args mock.Arguments) {
			capturedMsg = args.Get(1).(*ws.Message)
		}).Return()

	// Act
	_, err := service.GetConversation(ctx, userID, rideID, 50, 0)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, capturedMsg)
	assert.Equal(t, "chat.delivered", capturedMsg.Type)
	assert.Equal(t, rideID.String(), capturedMsg.Data["ride_id"])
	assert.Equal(t, userID.String(), capturedMsg.Data["delivered_to"])
	mockHub.AssertExpectations(t)
}

func TestService_GetConversation_NoHubNoDeliveryNotification(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	messages := []ChatMessage{{ID: uuid.New()}}

	mockRepo.On("GetMessagesByRide", ctx, rideID, 50, 0).Return(messages, nil)
	mockRepo.On("GetUnreadCount", ctx, rideID, userID).Return(0, nil)
	mockRepo.On("GetLastMessage", ctx, rideID).Return(nil, nil)
	mockRepo.On("MarkMessagesDelivered", ctx, rideID, userID).Return(nil)

	// Act
	conversation, err := service.GetConversation(ctx, userID, rideID, 50, 0)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, conversation)
	mockRepo.AssertExpectations(t)
}

// ===== MarkAsRead Tests =====

func TestService_MarkAsRead_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()
	lastReadID := uuid.New()

	req := &MarkReadRequest{
		RideID:     rideID,
		LastReadID: lastReadID,
	}

	mockRepo.On("MarkMessagesRead", ctx, rideID, userID, lastReadID).Return(nil)

	var capturedMsg *ws.Message
	mockHub.On("SendToRide", rideID.String(), mock.AnythingOfType("*websocket.Message")).
		Run(func(args mock.Arguments) {
			capturedMsg = args.Get(1).(*ws.Message)
		}).Return()

	// Act
	err := service.MarkAsRead(ctx, userID, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, capturedMsg)
	assert.Equal(t, "chat.read", capturedMsg.Type)
	assert.Equal(t, rideID.String(), capturedMsg.Data["ride_id"])
	assert.Equal(t, userID.String(), capturedMsg.Data["read_by"])
	assert.Equal(t, lastReadID.String(), capturedMsg.Data["last_read_id"])
	mockRepo.AssertExpectations(t)
	mockHub.AssertExpectations(t)
}

func TestService_MarkAsRead_RepoError(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	mockHub := new(MockChatHub)
	service := NewService(mockRepo, mockHub)

	ctx := context.Background()
	userID := uuid.New()

	req := &MarkReadRequest{
		RideID:     uuid.New(),
		LastReadID: uuid.New(),
	}

	mockRepo.On("MarkMessagesRead", ctx, req.RideID, userID, req.LastReadID).
		Return(errors.New("database error"))

	// Act
	err := service.MarkAsRead(ctx, userID, req)

	// Assert
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_MarkAsRead_NoHub(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()

	req := &MarkReadRequest{
		RideID:     uuid.New(),
		LastReadID: uuid.New(),
	}

	mockRepo.On("MarkMessagesRead", ctx, req.RideID, userID, req.LastReadID).Return(nil)

	// Act
	err := service.MarkAsRead(ctx, userID, req)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ===== GetQuickReplies Tests =====

func TestService_GetQuickReplies_Success(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		replies []QuickReply
	}{
		{
			name: "rider quick replies",
			role: "rider",
			replies: []QuickReply{
				{ID: uuid.New(), Role: "rider", Category: "eta", Text: "Where are you?", SortOrder: 1, IsActive: true},
				{ID: uuid.New(), Role: "rider", Category: "location", Text: "I'm waiting outside", SortOrder: 2, IsActive: true},
			},
		},
		{
			name: "driver quick replies",
			role: "driver",
			replies: []QuickReply{
				{ID: uuid.New(), Role: "driver", Category: "eta", Text: "I'll be there in 5 minutes", SortOrder: 1, IsActive: true},
				{ID: uuid.New(), Role: "driver", Category: "arrival", Text: "I've arrived", SortOrder: 2, IsActive: true},
			},
		},
		{
			name:    "empty quick replies",
			role:    "rider",
			replies: []QuickReply{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := new(MockChatRepository)
			service := NewService(mockRepo, nil)

			ctx := context.Background()

			mockRepo.On("GetQuickReplies", ctx, tt.role).Return(tt.replies, nil)

			// Act
			result, err := service.GetQuickReplies(ctx, tt.role)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.replies, result)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_GetQuickReplies_NoRows(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil)

	ctx := context.Background()

	mockRepo.On("GetQuickReplies", ctx, "rider").Return(nil, pgx.ErrNoRows)

	// Act
	result, err := service.GetQuickReplies(ctx, "rider")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
	mockRepo.AssertExpectations(t)
}

func TestService_GetQuickReplies_NilResult(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil)

	ctx := context.Background()

	mockRepo.On("GetQuickReplies", ctx, "driver").Return(nil, nil)

	// Act
	result, err := service.GetQuickReplies(ctx, "driver")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
	mockRepo.AssertExpectations(t)
}

func TestService_GetQuickReplies_Error(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil)

	ctx := context.Background()

	mockRepo.On("GetQuickReplies", ctx, "rider").Return(nil, errors.New("database error"))

	// Act
	result, err := service.GetQuickReplies(ctx, "rider")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

// ===== GetActiveConversations Tests =====

func TestService_GetActiveConversations_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()

	rideIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	mockRepo.On("GetActiveConversations", ctx, userID).Return(rideIDs, nil)

	// Act
	result, err := service.GetActiveConversations(ctx, userID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, rideIDs, result)
	mockRepo.AssertExpectations(t)
}

func TestService_GetActiveConversations_Empty(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("GetActiveConversations", ctx, userID).Return([]uuid.UUID{}, nil)

	// Act
	result, err := service.GetActiveConversations(ctx, userID)

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, result)
	mockRepo.AssertExpectations(t)
}

func TestService_GetActiveConversations_Error(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("GetActiveConversations", ctx, userID).Return(nil, errors.New("database error"))

	// Act
	result, err := service.GetActiveConversations(ctx, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

// ===== CleanupOldMessages Tests =====

func TestService_CleanupOldMessages_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil)

	ctx := context.Background()
	rideID := uuid.New()

	mockRepo.On("DeleteMessagesByRide", ctx, rideID).Return(nil)

	// Act
	err := service.CleanupOldMessages(ctx, rideID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_CleanupOldMessages_Error(t *testing.T) {
	// Arrange
	mockRepo := new(MockChatRepository)
	service := NewService(mockRepo, nil)

	ctx := context.Background()
	rideID := uuid.New()

	mockRepo.On("DeleteMessagesByRide", ctx, rideID).Return(errors.New("database error"))

	// Act
	err := service.CleanupOldMessages(ctx, rideID)

	// Assert
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

// ===== isValidMessageType Tests =====

func TestIsValidMessageType(t *testing.T) {
	tests := []struct {
		messageType MessageType
		expected    bool
	}{
		{MessageTypeText, true},
		{MessageTypeImage, true},
		{MessageTypeLocation, true},
		{MessageTypeSystem, false}, // System messages are not valid for user-sent messages
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.messageType), func(t *testing.T) {
			result := isValidMessageType(tt.messageType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ===== Message Status Flow Tests =====

func TestService_MessageStatusFlow(t *testing.T) {
	// Test that messages go through the correct status flow: sent -> delivered -> read

	t.Run("message sent with status sent", func(t *testing.T) {
		mockRepo := new(MockChatRepository)
		mockHub := new(MockChatHub)
		service := NewService(mockRepo, mockHub)

		ctx := context.Background()
		senderID := uuid.New()

		req := &SendMessageRequest{
			RideID:      uuid.New(),
			MessageType: MessageTypeText,
			Content:     "Hello",
		}

		var savedMsg *ChatMessage
		mockRepo.On("SaveMessage", ctx, mock.AnythingOfType("*chat.ChatMessage")).
			Run(func(args mock.Arguments) {
				savedMsg = args.Get(1).(*ChatMessage)
			}).Return(nil)
		mockHub.On("SendToRide", mock.Anything, mock.Anything).Return()

		msg, err := service.SendMessage(ctx, senderID, "rider", req)

		assert.NoError(t, err)
		assert.Equal(t, MessageStatusSent, msg.Status)
		assert.Equal(t, MessageStatusSent, savedMsg.Status)
	})
}

// ===== Helper Functions =====

func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}
