package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/chat"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
	"github.com/stretchr/testify/mock"
)

// MockChatRepository is a mock implementation of chat.RepositoryInterface
type MockChatRepository struct {
	mock.Mock
}

func (m *MockChatRepository) SaveMessage(ctx context.Context, msg *chat.ChatMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockChatRepository) GetMessagesByRide(ctx context.Context, rideID uuid.UUID, limit, offset int) ([]chat.ChatMessage, error) {
	args := m.Called(ctx, rideID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]chat.ChatMessage), args.Error(1)
}

func (m *MockChatRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*chat.ChatMessage, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*chat.ChatMessage), args.Error(1)
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

func (m *MockChatRepository) GetLastMessage(ctx context.Context, rideID uuid.UUID) (*chat.ChatMessage, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*chat.ChatMessage), args.Error(1)
}

func (m *MockChatRepository) DeleteMessagesByRide(ctx context.Context, rideID uuid.UUID) error {
	args := m.Called(ctx, rideID)
	return args.Error(0)
}

func (m *MockChatRepository) GetMessageCount(ctx context.Context, rideID uuid.UUID) (int, error) {
	args := m.Called(ctx, rideID)
	return args.Int(0), args.Error(1)
}

func (m *MockChatRepository) GetQuickReplies(ctx context.Context, role string) ([]chat.QuickReply, error) {
	args := m.Called(ctx, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]chat.QuickReply), args.Error(1)
}

func (m *MockChatRepository) CreateQuickReply(ctx context.Context, qr *chat.QuickReply) error {
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

// MockChatHub is a mock implementation of chat.HubInterface
type MockChatHub struct {
	mock.Mock
}

func (m *MockChatHub) SendToRide(rideID string, msg *ws.Message) {
	m.Called(rideID, msg)
}

func (m *MockChatHub) SendToUser(userID string, msg *ws.Message) {
	m.Called(userID, msg)
}
