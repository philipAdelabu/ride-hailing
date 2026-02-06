package chat

import (
	"context"

	"github.com/google/uuid"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
)

// RepositoryInterface defines all repository operations for chat
type RepositoryInterface interface {
	// Message operations
	SaveMessage(ctx context.Context, msg *ChatMessage) error
	GetMessagesByRide(ctx context.Context, rideID uuid.UUID, limit, offset int) ([]ChatMessage, error)
	GetMessageByID(ctx context.Context, messageID uuid.UUID) (*ChatMessage, error)
	MarkMessagesDelivered(ctx context.Context, rideID, recipientID uuid.UUID) error
	MarkMessagesRead(ctx context.Context, rideID, recipientID, lastReadID uuid.UUID) error
	GetUnreadCount(ctx context.Context, rideID, userID uuid.UUID) (int, error)
	GetLastMessage(ctx context.Context, rideID uuid.UUID) (*ChatMessage, error)
	DeleteMessagesByRide(ctx context.Context, rideID uuid.UUID) error
	GetMessageCount(ctx context.Context, rideID uuid.UUID) (int, error)

	// Quick replies
	GetQuickReplies(ctx context.Context, role string) ([]QuickReply, error)
	CreateQuickReply(ctx context.Context, qr *QuickReply) error

	// Active conversations
	GetActiveConversations(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

// HubInterface defines the WebSocket hub operations used by the chat service
type HubInterface interface {
	SendToRide(rideID string, msg *ws.Message)
	SendToUser(userID string, msg *ws.Message)
}
