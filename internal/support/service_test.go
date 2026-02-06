package support

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockRepo is a test-local mock that implements RepositoryInterface.
type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) CreateTicket(ctx context.Context, ticket *Ticket) error {
	args := m.Called(ctx, ticket)
	return args.Error(0)
}

func (m *mockRepo) GetTicketByID(ctx context.Context, id uuid.UUID) (*Ticket, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Ticket), args.Error(1)
}

func (m *mockRepo) GetUserTickets(ctx context.Context, userID uuid.UUID, status *TicketStatus, limit, offset int) ([]TicketSummary, int, error) {
	args := m.Called(ctx, userID, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]TicketSummary), args.Int(1), args.Error(2)
}

func (m *mockRepo) GetAllTickets(ctx context.Context, status *TicketStatus, priority *TicketPriority, category *TicketCategory, limit, offset int) ([]TicketSummary, int, error) {
	args := m.Called(ctx, status, priority, category, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]TicketSummary), args.Int(1), args.Error(2)
}

func (m *mockRepo) UpdateTicket(ctx context.Context, id uuid.UUID, status *TicketStatus, priority *TicketPriority, assignedTo *uuid.UUID, tags []string) error {
	args := m.Called(ctx, id, status, priority, assignedTo, tags)
	return args.Error(0)
}

func (m *mockRepo) GetTicketStats(ctx context.Context) (*TicketStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TicketStats), args.Error(1)
}

func (m *mockRepo) CreateMessage(ctx context.Context, msg *TicketMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockRepo) GetMessagesByTicket(ctx context.Context, ticketID uuid.UUID, includeInternal bool) ([]TicketMessage, error) {
	args := m.Called(ctx, ticketID, includeInternal)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TicketMessage), args.Error(1)
}

func (m *mockRepo) GetFAQArticles(ctx context.Context, category *string) ([]FAQArticle, error) {
	args := m.Called(ctx, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]FAQArticle), args.Error(1)
}

func (m *mockRepo) GetFAQArticleByID(ctx context.Context, id uuid.UUID) (*FAQArticle, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FAQArticle), args.Error(1)
}

func (m *mockRepo) IncrementFAQViewCount(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// newTestService creates a Service wired to the given mock repo for testing.
func newTestService(repo *mockRepo) *Service {
	return &Service{
		repo: repo,
	}
}

// ========================================
// CREATE TICKET TESTS
// ========================================

func TestCreateTicket_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateTicket", ctx, mock.AnythingOfType("*support.Ticket")).Return(nil)
	repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)

	req := &CreateTicketRequest{
		Category:    CategoryPayment,
		Subject:     "Payment issue",
		Description: "I was charged twice for my ride",
	}

	ticket, err := svc.CreateTicket(ctx, userID, req)
	require.NoError(t, err)
	require.NotNil(t, ticket)

	assert.NotEqual(t, uuid.Nil, ticket.ID)
	assert.Equal(t, userID, ticket.UserID)
	assert.Equal(t, CategoryPayment, ticket.Category)
	assert.Equal(t, PriorityMedium, ticket.Priority) // Default priority
	assert.Equal(t, TicketStatusOpen, ticket.Status)
	assert.Equal(t, "Payment issue", ticket.Subject)
	assert.Equal(t, "I was charged twice for my ride", ticket.Description)
	assert.NotEmpty(t, ticket.TicketNumber)

	repo.AssertExpectations(t)
}

func TestCreateTicket_WithPriority(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateTicket", ctx, mock.AnythingOfType("*support.Ticket")).Return(nil)
	repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)

	highPriority := PriorityHigh
	req := &CreateTicketRequest{
		Category:    CategoryRide,
		Subject:     "Driver issue",
		Description: "Driver was rude and unprofessional",
		Priority:    &highPriority,
	}

	ticket, err := svc.CreateTicket(ctx, userID, req)
	require.NoError(t, err)
	require.NotNil(t, ticket)

	assert.Equal(t, PriorityHigh, ticket.Priority)

	repo.AssertExpectations(t)
}

func TestCreateTicket_SafetyCategoryAutoEscalates(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateTicket", ctx, mock.AnythingOfType("*support.Ticket")).Return(nil)
	repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)

	// Even if low priority is requested, safety issues should auto-escalate to urgent
	lowPriority := PriorityLow
	req := &CreateTicketRequest{
		Category:    CategorySafety,
		Subject:     "Safety concern",
		Description: "I felt unsafe during my ride due to driver behavior",
		Priority:    &lowPriority,
	}

	ticket, err := svc.CreateTicket(ctx, userID, req)
	require.NoError(t, err)
	require.NotNil(t, ticket)

	// Safety category should auto-escalate to urgent priority
	assert.Equal(t, PriorityUrgent, ticket.Priority)
	assert.Equal(t, CategorySafety, ticket.Category)

	repo.AssertExpectations(t)
}

func TestCreateTicket_WithRideID(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	repo.On("CreateTicket", ctx, mock.AnythingOfType("*support.Ticket")).Return(nil)
	repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)

	req := &CreateTicketRequest{
		Category:    CategoryFareDispute,
		Subject:     "Fare dispute",
		Description: "I was overcharged for this specific ride",
		RideID:      &rideID,
	}

	ticket, err := svc.CreateTicket(ctx, userID, req)
	require.NoError(t, err)
	require.NotNil(t, ticket)

	assert.Equal(t, &rideID, ticket.RideID)

	repo.AssertExpectations(t)
}

func TestCreateTicket_RepoError(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateTicket", ctx, mock.AnythingOfType("*support.Ticket")).Return(errors.New("db connection failed"))

	req := &CreateTicketRequest{
		Category:    CategoryPayment,
		Subject:     "Payment issue",
		Description: "I was charged twice for my ride",
	}

	ticket, err := svc.CreateTicket(ctx, userID, req)
	require.Error(t, err)
	assert.Nil(t, ticket)
	assert.Contains(t, err.Error(), "create ticket")

	repo.AssertExpectations(t)
}

// ========================================
// GET MY TICKETS TESTS
// ========================================

func TestGetMyTickets_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	expected := []TicketSummary{
		{
			ID:           uuid.New(),
			TicketNumber: "TKT-000001",
			Category:     CategoryPayment,
			Priority:     PriorityMedium,
			Status:       TicketStatusOpen,
			Subject:      "Payment issue",
			MessageCount: 2,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	repo.On("GetUserTickets", ctx, userID, (*TicketStatus)(nil), 20, 0).Return(expected, 1, nil)

	tickets, total, err := svc.GetMyTickets(ctx, userID, nil, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, tickets, 1)
	assert.Equal(t, "TKT-000001", tickets[0].TicketNumber)

	repo.AssertExpectations(t)
}

func TestGetMyTickets_WithStatusFilter(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	openStatus := TicketStatusOpen
	expected := []TicketSummary{
		{
			ID:           uuid.New(),
			TicketNumber: "TKT-000001",
			Category:     CategoryPayment,
			Priority:     PriorityMedium,
			Status:       TicketStatusOpen,
			Subject:      "Open ticket",
		},
	}

	repo.On("GetUserTickets", ctx, userID, &openStatus, 20, 0).Return(expected, 1, nil)

	tickets, total, err := svc.GetMyTickets(ctx, userID, &openStatus, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, tickets, 1)

	repo.AssertExpectations(t)
}

func TestGetMyTickets_Pagination(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		expectedSize int
		expectedOff  int
	}{
		{
			name:         "page 1 with default size",
			page:         1,
			pageSize:     20,
			expectedSize: 20,
			expectedOff:  0,
		},
		{
			name:         "page 2 with size 10",
			page:         2,
			pageSize:     10,
			expectedSize: 10,
			expectedOff:  10,
		},
		{
			name:         "page 0 defaults to 1",
			page:         0,
			pageSize:     20,
			expectedSize: 20,
			expectedOff:  0,
		},
		{
			name:         "negative page defaults to 1",
			page:         -1,
			pageSize:     20,
			expectedSize: 20,
			expectedOff:  0,
		},
		{
			name:         "pageSize 0 defaults to 20",
			page:         1,
			pageSize:     0,
			expectedSize: 20,
			expectedOff:  0,
		},
		{
			name:         "pageSize > 50 caps at 20",
			page:         1,
			pageSize:     100,
			expectedSize: 20,
			expectedOff:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			svc := newTestService(repo)
			ctx := context.Background()
			userID := uuid.New()

			repo.On("GetUserTickets", ctx, userID, (*TicketStatus)(nil), tt.expectedSize, tt.expectedOff).
				Return([]TicketSummary{}, 0, nil)

			_, _, err := svc.GetMyTickets(ctx, userID, nil, tt.page, tt.pageSize)
			require.NoError(t, err)

			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// GET TICKET TESTS
// ========================================

func TestGetTicket_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	expected := &Ticket{
		ID:           ticketID,
		UserID:       userID,
		TicketNumber: "TKT-000001",
		Category:     CategoryPayment,
		Priority:     PriorityMedium,
		Status:       TicketStatusOpen,
		Subject:      "Payment issue",
		Description:  "I was charged twice",
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(expected, nil)

	ticket, err := svc.GetTicket(ctx, ticketID, userID)
	require.NoError(t, err)
	require.NotNil(t, ticket)

	assert.Equal(t, ticketID, ticket.ID)
	assert.Equal(t, userID, ticket.UserID)

	repo.AssertExpectations(t)
}

func TestGetTicket_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	repo.On("GetTicketByID", ctx, ticketID).Return(nil, pgx.ErrNoRows)

	ticket, err := svc.GetTicket(ctx, ticketID, userID)
	require.Error(t, err)
	assert.Nil(t, ticket)
	assert.Contains(t, err.Error(), "ticket not found")

	repo.AssertExpectations(t)
}

func TestGetTicket_Forbidden(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: otherUserID, // Different user
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)

	result, err := svc.GetTicket(ctx, ticketID, userID)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "forbidden")

	repo.AssertExpectations(t)
}

// ========================================
// ADD MESSAGE TESTS
// ========================================

func TestAddMessage_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: userID,
		Status: TicketStatusOpen,
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)
	repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)

	req := &AddMessageRequest{
		Message: "Here is more information about my issue",
	}

	msg, err := svc.AddMessage(ctx, ticketID, userID, req)
	require.NoError(t, err)
	require.NotNil(t, msg)

	assert.NotEqual(t, uuid.Nil, msg.ID)
	assert.Equal(t, ticketID, msg.TicketID)
	assert.Equal(t, userID, msg.SenderID)
	assert.Equal(t, SenderUser, msg.SenderType)
	assert.Equal(t, "Here is more information about my issue", msg.Message)
	assert.False(t, msg.IsInternal)

	repo.AssertExpectations(t)
}

func TestAddMessage_WithAttachments(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: userID,
		Status: TicketStatusOpen,
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)
	repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)

	req := &AddMessageRequest{
		Message:     "Here is a screenshot of the issue",
		Attachments: []string{"https://storage.example.com/screenshot1.png"},
	}

	msg, err := svc.AddMessage(ctx, ticketID, userID, req)
	require.NoError(t, err)
	require.NotNil(t, msg)

	assert.Len(t, msg.Attachments, 1)
	assert.Equal(t, "https://storage.example.com/screenshot1.png", msg.Attachments[0])

	repo.AssertExpectations(t)
}

func TestAddMessage_TicketNotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	repo.On("GetTicketByID", ctx, ticketID).Return(nil, pgx.ErrNoRows)

	req := &AddMessageRequest{
		Message: "Message to add",
	}

	msg, err := svc.AddMessage(ctx, ticketID, userID, req)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "ticket not found")

	repo.AssertExpectations(t)
}

func TestAddMessage_NotAuthorized(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: otherUserID,
		Status: TicketStatusOpen,
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)

	req := &AddMessageRequest{
		Message: "Message to add",
	}

	msg, err := svc.AddMessage(ctx, ticketID, userID, req)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "forbidden")

	repo.AssertExpectations(t)
}

func TestAddMessage_ClosedTicket(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: userID,
		Status: TicketStatusClosed,
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)

	req := &AddMessageRequest{
		Message: "Message to add",
	}

	msg, err := svc.AddMessage(ctx, ticketID, userID, req)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "cannot reply to a closed ticket")

	repo.AssertExpectations(t)
}

func TestAddMessage_ReopensWaitingTicket(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: userID,
		Status: TicketStatusWaiting,
	}

	openStatus := TicketStatusOpen
	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)
	repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)
	repo.On("UpdateTicket", ctx, ticketID, &openStatus, (*TicketPriority)(nil), (*uuid.UUID)(nil), ([]string)(nil)).Return(nil)

	req := &AddMessageRequest{
		Message: "Additional information",
	}

	msg, err := svc.AddMessage(ctx, ticketID, userID, req)
	require.NoError(t, err)
	require.NotNil(t, msg)

	repo.AssertExpectations(t)
}

func TestAddMessage_ReopensResolvedTicket(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: userID,
		Status: TicketStatusResolved,
	}

	openStatus := TicketStatusOpen
	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)
	repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)
	repo.On("UpdateTicket", ctx, ticketID, &openStatus, (*TicketPriority)(nil), (*uuid.UUID)(nil), ([]string)(nil)).Return(nil)

	req := &AddMessageRequest{
		Message: "Issue not resolved",
	}

	msg, err := svc.AddMessage(ctx, ticketID, userID, req)
	require.NoError(t, err)
	require.NotNil(t, msg)

	repo.AssertExpectations(t)
}

// ========================================
// CLOSE TICKET TESTS
// ========================================

func TestCloseTicket_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: userID,
		Status: TicketStatusResolved,
	}

	closedStatus := TicketStatusClosed
	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)
	repo.On("UpdateTicket", ctx, ticketID, &closedStatus, (*TicketPriority)(nil), (*uuid.UUID)(nil), ([]string)(nil)).Return(nil)

	err := svc.CloseTicket(ctx, ticketID, userID)
	require.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestCloseTicket_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	repo.On("GetTicketByID", ctx, ticketID).Return(nil, pgx.ErrNoRows)

	err := svc.CloseTicket(ctx, ticketID, userID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket not found")

	repo.AssertExpectations(t)
}

func TestCloseTicket_NotAuthorized(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: otherUserID,
		Status: TicketStatusOpen,
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)

	err := svc.CloseTicket(ctx, ticketID, userID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")

	repo.AssertExpectations(t)
}

func TestCloseTicket_AlreadyClosed(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: userID,
		Status: TicketStatusClosed,
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)

	err := svc.CloseTicket(ctx, ticketID, userID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket is already closed")

	repo.AssertExpectations(t)
}

// ========================================
// GET TICKET MESSAGES TESTS
// ========================================

func TestGetTicketMessages_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: userID,
	}

	expected := []TicketMessage{
		{
			ID:         uuid.New(),
			TicketID:   ticketID,
			SenderID:   userID,
			SenderType: SenderUser,
			Message:    "Initial message",
			IsInternal: false,
		},
		{
			ID:         uuid.New(),
			TicketID:   ticketID,
			SenderID:   uuid.New(),
			SenderType: SenderAgent,
			Message:    "Agent response",
			IsInternal: false,
		},
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)
	repo.On("GetMessagesByTicket", ctx, ticketID, false).Return(expected, nil)

	messages, err := svc.GetTicketMessages(ctx, ticketID, userID)
	require.NoError(t, err)
	assert.Len(t, messages, 2)

	repo.AssertExpectations(t)
}

func TestGetTicketMessages_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	ticketID := uuid.New()

	repo.On("GetTicketByID", ctx, ticketID).Return(nil, pgx.ErrNoRows)

	messages, err := svc.GetTicketMessages(ctx, ticketID, userID)
	require.Error(t, err)
	assert.Nil(t, messages)
	assert.Contains(t, err.Error(), "ticket not found")

	repo.AssertExpectations(t)
}

func TestGetTicketMessages_NotAuthorized(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		UserID: otherUserID,
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)

	messages, err := svc.GetTicketMessages(ctx, ticketID, userID)
	require.Error(t, err)
	assert.Nil(t, messages)
	assert.Contains(t, err.Error(), "forbidden")

	repo.AssertExpectations(t)
}

// ========================================
// FAQ TESTS
// ========================================

func TestGetFAQArticles_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()

	expected := []FAQArticle{
		{
			ID:       uuid.New(),
			Category: "payments",
			Title:    "How to add a payment method",
			Content:  "Go to Settings > Payment Methods...",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			Category: "rides",
			Title:    "How to request a ride",
			Content:  "Open the app and enter your destination...",
			IsActive: true,
		},
	}

	repo.On("GetFAQArticles", ctx, (*string)(nil)).Return(expected, nil)

	articles, err := svc.GetFAQArticles(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, articles, 2)

	repo.AssertExpectations(t)
}

func TestGetFAQArticles_WithCategory(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()

	category := "payments"
	expected := []FAQArticle{
		{
			ID:       uuid.New(),
			Category: "payments",
			Title:    "How to add a payment method",
			Content:  "Go to Settings > Payment Methods...",
			IsActive: true,
		},
	}

	repo.On("GetFAQArticles", ctx, &category).Return(expected, nil)

	articles, err := svc.GetFAQArticles(ctx, &category)
	require.NoError(t, err)
	assert.Len(t, articles, 1)
	assert.Equal(t, "payments", articles[0].Category)

	repo.AssertExpectations(t)
}

func TestGetFAQArticle_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	articleID := uuid.New()

	expected := &FAQArticle{
		ID:        articleID,
		Category:  "payments",
		Title:     "How to add a payment method",
		Content:   "Go to Settings > Payment Methods...",
		IsActive:  true,
		ViewCount: 100,
	}

	repo.On("GetFAQArticleByID", ctx, articleID).Return(expected, nil)
	repo.On("IncrementFAQViewCount", ctx, articleID).Return(nil)

	article, err := svc.GetFAQArticle(ctx, articleID)
	require.NoError(t, err)
	require.NotNil(t, article)

	assert.Equal(t, articleID, article.ID)
	assert.Equal(t, "How to add a payment method", article.Title)

	repo.AssertExpectations(t)
}

func TestGetFAQArticle_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	articleID := uuid.New()

	repo.On("GetFAQArticleByID", ctx, articleID).Return(nil, pgx.ErrNoRows)

	article, err := svc.GetFAQArticle(ctx, articleID)
	require.Error(t, err)
	assert.Nil(t, article)
	assert.Contains(t, err.Error(), "article not found")

	repo.AssertExpectations(t)
}

// ========================================
// ADMIN TESTS
// ========================================

func TestAdminGetTickets_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()

	expected := []TicketSummary{
		{
			ID:           uuid.New(),
			TicketNumber: "TKT-000001",
			Category:     CategorySafety,
			Priority:     PriorityUrgent,
			Status:       TicketStatusOpen,
			Subject:      "Safety concern",
		},
		{
			ID:           uuid.New(),
			TicketNumber: "TKT-000002",
			Category:     CategoryPayment,
			Priority:     PriorityMedium,
			Status:       TicketStatusInProgress,
			Subject:      "Payment issue",
		},
	}

	repo.On("GetAllTickets", ctx, (*TicketStatus)(nil), (*TicketPriority)(nil), (*TicketCategory)(nil), 20, 0).
		Return(expected, 2, nil)

	tickets, total, err := svc.AdminGetTickets(ctx, nil, nil, nil, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, tickets, 2)

	repo.AssertExpectations(t)
}

func TestAdminGetTickets_WithFilters(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()

	status := TicketStatusOpen
	priority := PriorityUrgent
	category := CategorySafety

	expected := []TicketSummary{
		{
			ID:           uuid.New(),
			TicketNumber: "TKT-000001",
			Category:     CategorySafety,
			Priority:     PriorityUrgent,
			Status:       TicketStatusOpen,
			Subject:      "Safety concern",
		},
	}

	repo.On("GetAllTickets", ctx, &status, &priority, &category, 20, 0).
		Return(expected, 1, nil)

	tickets, total, err := svc.AdminGetTickets(ctx, &status, &priority, &category, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, tickets, 1)

	repo.AssertExpectations(t)
}

func TestAdminGetTickets_PaginationDefaults(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		expectedSize int
		expectedOff  int
	}{
		{
			name:         "page 0 defaults to 1",
			page:         0,
			pageSize:     20,
			expectedSize: 20,
			expectedOff:  0,
		},
		{
			name:         "pageSize 0 defaults to 20",
			page:         1,
			pageSize:     0,
			expectedSize: 20,
			expectedOff:  0,
		},
		{
			name:         "pageSize > 100 caps at 20",
			page:         1,
			pageSize:     150,
			expectedSize: 20,
			expectedOff:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			svc := newTestService(repo)
			ctx := context.Background()

			repo.On("GetAllTickets", ctx, (*TicketStatus)(nil), (*TicketPriority)(nil), (*TicketCategory)(nil), tt.expectedSize, tt.expectedOff).
				Return([]TicketSummary{}, 0, nil)

			_, _, err := svc.AdminGetTickets(ctx, nil, nil, nil, tt.page, tt.pageSize)
			require.NoError(t, err)

			repo.AssertExpectations(t)
		})
	}
}

func TestAdminGetTicket_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	ticketID := uuid.New()

	expected := &Ticket{
		ID:           ticketID,
		UserID:       uuid.New(),
		TicketNumber: "TKT-000001",
		Category:     CategorySafety,
		Priority:     PriorityUrgent,
		Status:       TicketStatusOpen,
		Subject:      "Safety concern",
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(expected, nil)

	ticket, err := svc.AdminGetTicket(ctx, ticketID)
	require.NoError(t, err)
	require.NotNil(t, ticket)

	assert.Equal(t, ticketID, ticket.ID)

	repo.AssertExpectations(t)
}

func TestAdminGetTicket_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	ticketID := uuid.New()

	repo.On("GetTicketByID", ctx, ticketID).Return(nil, pgx.ErrNoRows)

	ticket, err := svc.AdminGetTicket(ctx, ticketID)
	require.Error(t, err)
	assert.Nil(t, ticket)
	assert.Contains(t, err.Error(), "ticket not found")

	repo.AssertExpectations(t)
}

func TestAdminGetMessages_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID: ticketID,
	}

	expected := []TicketMessage{
		{
			ID:         uuid.New(),
			TicketID:   ticketID,
			SenderID:   uuid.New(),
			SenderType: SenderUser,
			Message:    "User message",
			IsInternal: false,
		},
		{
			ID:         uuid.New(),
			TicketID:   ticketID,
			SenderID:   uuid.New(),
			SenderType: SenderAgent,
			Message:    "Internal note - escalating",
			IsInternal: true,
		},
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)
	repo.On("GetMessagesByTicket", ctx, ticketID, true).Return(expected, nil) // includeInternal = true

	messages, err := svc.AdminGetMessages(ctx, ticketID)
	require.NoError(t, err)
	assert.Len(t, messages, 2)

	// Verify internal messages are included
	hasInternal := false
	for _, msg := range messages {
		if msg.IsInternal {
			hasInternal = true
			break
		}
	}
	assert.True(t, hasInternal, "Admin should see internal messages")

	repo.AssertExpectations(t)
}

func TestAdminGetMessages_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	ticketID := uuid.New()

	repo.On("GetTicketByID", ctx, ticketID).Return(nil, pgx.ErrNoRows)

	messages, err := svc.AdminGetMessages(ctx, ticketID)
	require.Error(t, err)
	assert.Nil(t, messages)
	assert.Contains(t, err.Error(), "ticket not found")

	repo.AssertExpectations(t)
}

func TestAdminReply_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	ticketID := uuid.New()
	agentID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		Status: TicketStatusOpen,
	}

	waitingStatus := TicketStatusWaiting
	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)
	repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)
	repo.On("UpdateTicket", ctx, ticketID, &waitingStatus, (*TicketPriority)(nil), &agentID, ([]string)(nil)).Return(nil)

	req := &AdminReplyRequest{
		Message:    "Thank you for contacting support. We are looking into this.",
		IsInternal: false,
	}

	msg, err := svc.AdminReply(ctx, ticketID, agentID, req)
	require.NoError(t, err)
	require.NotNil(t, msg)

	assert.Equal(t, ticketID, msg.TicketID)
	assert.Equal(t, agentID, msg.SenderID)
	assert.Equal(t, SenderAgent, msg.SenderType)
	assert.False(t, msg.IsInternal)

	repo.AssertExpectations(t)
}

func TestAdminReply_InternalNote(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	ticketID := uuid.New()
	agentID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		Status: TicketStatusOpen,
	}

	inProgressStatus := TicketStatusInProgress
	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)
	repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)
	repo.On("UpdateTicket", ctx, ticketID, &inProgressStatus, (*TicketPriority)(nil), &agentID, ([]string)(nil)).Return(nil)

	req := &AdminReplyRequest{
		Message:    "Escalating this to senior support",
		IsInternal: true,
	}

	msg, err := svc.AdminReply(ctx, ticketID, agentID, req)
	require.NoError(t, err)
	require.NotNil(t, msg)

	assert.True(t, msg.IsInternal)

	repo.AssertExpectations(t)
}

func TestAdminReply_ClosedTicket(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	ticketID := uuid.New()
	agentID := uuid.New()

	ticket := &Ticket{
		ID:     ticketID,
		Status: TicketStatusClosed,
	}

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)

	req := &AdminReplyRequest{
		Message: "Reply to closed ticket",
	}

	msg, err := svc.AdminReply(ctx, ticketID, agentID, req)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "cannot reply to a closed ticket")

	repo.AssertExpectations(t)
}

func TestAdminReply_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	ticketID := uuid.New()
	agentID := uuid.New()

	repo.On("GetTicketByID", ctx, ticketID).Return(nil, pgx.ErrNoRows)

	req := &AdminReplyRequest{
		Message: "Reply",
	}

	msg, err := svc.AdminReply(ctx, ticketID, agentID, req)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "ticket not found")

	repo.AssertExpectations(t)
}

func TestAdminUpdateTicket_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	ticketID := uuid.New()

	ticket := &Ticket{
		ID:       ticketID,
		Status:   TicketStatusOpen,
		Priority: PriorityMedium,
	}

	escalatedStatus := TicketStatusEscalated
	urgentPriority := PriorityUrgent
	agentID := uuid.New()

	repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)
	repo.On("UpdateTicket", ctx, ticketID, &escalatedStatus, &urgentPriority, &agentID, []string{"safety", "urgent"}).Return(nil)

	req := &UpdateTicketRequest{
		Status:     &escalatedStatus,
		Priority:   &urgentPriority,
		AssignedTo: &agentID,
		Tags:       []string{"safety", "urgent"},
	}

	err := svc.AdminUpdateTicket(ctx, ticketID, req)
	require.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestAdminUpdateTicket_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	ticketID := uuid.New()

	repo.On("GetTicketByID", ctx, ticketID).Return(nil, pgx.ErrNoRows)

	req := &UpdateTicketRequest{
		Tags: []string{"test"},
	}

	err := svc.AdminUpdateTicket(ctx, ticketID, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket not found")

	repo.AssertExpectations(t)
}

func TestAdminGetStats_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()

	expected := &TicketStats{
		TotalOpen:        10,
		TotalInProgress:  5,
		TotalWaiting:     3,
		TotalResolved:    100,
		TotalEscalated:   2,
		AvgResolutionMin: 45.5,
		ByCategory: map[string]int{
			"payment": 5,
			"safety":  3,
			"ride":    7,
		},
		ByPriority: map[string]int{
			"low":    3,
			"medium": 8,
			"high":   4,
			"urgent": 3,
		},
	}

	repo.On("GetTicketStats", ctx).Return(expected, nil)

	stats, err := svc.AdminGetStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, 10, stats.TotalOpen)
	assert.Equal(t, 5, stats.TotalInProgress)
	assert.Equal(t, 3, stats.TotalWaiting)
	assert.Equal(t, 100, stats.TotalResolved)
	assert.Equal(t, 2, stats.TotalEscalated)
	assert.InDelta(t, 45.5, stats.AvgResolutionMin, 0.01)
	assert.Equal(t, 5, stats.ByCategory["payment"])
	assert.Equal(t, 3, stats.ByPriority["urgent"])

	repo.AssertExpectations(t)
}

// ========================================
// SAFETY ESCALATION TESTS (Critical Path)
// ========================================

func TestCreateTicket_SafetyCategories_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		category         TicketCategory
		requestPriority  *TicketPriority
		expectedPriority TicketPriority
	}{
		{
			name:             "safety category with no priority auto-escalates to urgent",
			category:         CategorySafety,
			requestPriority:  nil,
			expectedPriority: PriorityUrgent,
		},
		{
			name:             "safety category with low priority auto-escalates to urgent",
			category:         CategorySafety,
			requestPriority:  ptrPriority(PriorityLow),
			expectedPriority: PriorityUrgent,
		},
		{
			name:             "safety category with medium priority auto-escalates to urgent",
			category:         CategorySafety,
			requestPriority:  ptrPriority(PriorityMedium),
			expectedPriority: PriorityUrgent,
		},
		{
			name:             "safety category with high priority auto-escalates to urgent",
			category:         CategorySafety,
			requestPriority:  ptrPriority(PriorityHigh),
			expectedPriority: PriorityUrgent,
		},
		{
			name:             "payment category with no priority uses default medium",
			category:         CategoryPayment,
			requestPriority:  nil,
			expectedPriority: PriorityMedium,
		},
		{
			name:             "payment category respects requested high priority",
			category:         CategoryPayment,
			requestPriority:  ptrPriority(PriorityHigh),
			expectedPriority: PriorityHigh,
		},
		{
			name:             "ride category respects requested low priority",
			category:         CategoryRide,
			requestPriority:  ptrPriority(PriorityLow),
			expectedPriority: PriorityLow,
		},
		{
			name:             "driver category uses default medium when no priority",
			category:         CategoryDriver,
			requestPriority:  nil,
			expectedPriority: PriorityMedium,
		},
		{
			name:             "lost item category with urgent priority uses urgent",
			category:         CategoryLostItem,
			requestPriority:  ptrPriority(PriorityUrgent),
			expectedPriority: PriorityUrgent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			svc := newTestService(repo)
			ctx := context.Background()
			userID := uuid.New()

			repo.On("CreateTicket", ctx, mock.AnythingOfType("*support.Ticket")).Return(nil)
			repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)

			req := &CreateTicketRequest{
				Category:    tt.category,
				Subject:     "Test ticket",
				Description: "This is a test ticket description",
				Priority:    tt.requestPriority,
			}

			ticket, err := svc.CreateTicket(ctx, userID, req)
			require.NoError(t, err)
			require.NotNil(t, ticket)

			assert.Equal(t, tt.expectedPriority, ticket.Priority,
				"Expected priority %s but got %s for category %s",
				tt.expectedPriority, ticket.Priority, tt.category)

			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// STATUS TRANSITION TESTS
// ========================================

func TestStatusTransitions_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus TicketStatus
		action        string
		expectError   bool
		errorContains string
	}{
		{
			name:          "user can close resolved ticket",
			initialStatus: TicketStatusResolved,
			action:        "close",
			expectError:   false,
		},
		{
			name:          "user can close open ticket",
			initialStatus: TicketStatusOpen,
			action:        "close",
			expectError:   false,
		},
		{
			name:          "user cannot close already closed ticket",
			initialStatus: TicketStatusClosed,
			action:        "close",
			expectError:   true,
			errorContains: "already closed",
		},
		{
			name:          "user can add message to open ticket",
			initialStatus: TicketStatusOpen,
			action:        "add_message",
			expectError:   false,
		},
		{
			name:          "user cannot add message to closed ticket",
			initialStatus: TicketStatusClosed,
			action:        "add_message",
			expectError:   true,
			errorContains: "cannot reply to a closed ticket",
		},
		{
			name:          "user can add message to waiting ticket (reopens)",
			initialStatus: TicketStatusWaiting,
			action:        "add_message",
			expectError:   false,
		},
		{
			name:          "user can add message to resolved ticket (reopens)",
			initialStatus: TicketStatusResolved,
			action:        "add_message",
			expectError:   false,
		},
		{
			name:          "user can add message to in_progress ticket",
			initialStatus: TicketStatusInProgress,
			action:        "add_message",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			svc := newTestService(repo)
			ctx := context.Background()
			userID := uuid.New()
			ticketID := uuid.New()

			ticket := &Ticket{
				ID:     ticketID,
				UserID: userID,
				Status: tt.initialStatus,
			}

			repo.On("GetTicketByID", ctx, ticketID).Return(ticket, nil)

			switch tt.action {
			case "close":
				if !tt.expectError {
					closedStatus := TicketStatusClosed
					repo.On("UpdateTicket", ctx, ticketID, &closedStatus, (*TicketPriority)(nil), (*uuid.UUID)(nil), ([]string)(nil)).Return(nil)
				}

				err := svc.CloseTicket(ctx, ticketID, userID)

				if tt.expectError {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tt.errorContains)
				} else {
					require.NoError(t, err)
				}

			case "add_message":
				if !tt.expectError {
					repo.On("CreateMessage", ctx, mock.AnythingOfType("*support.TicketMessage")).Return(nil)
					if tt.initialStatus == TicketStatusWaiting || tt.initialStatus == TicketStatusResolved {
						openStatus := TicketStatusOpen
						repo.On("UpdateTicket", ctx, ticketID, &openStatus, (*TicketPriority)(nil), (*uuid.UUID)(nil), ([]string)(nil)).Return(nil)
					}
				}

				req := &AddMessageRequest{Message: "Test message"}
				_, err := svc.AddMessage(ctx, ticketID, userID, req)

				if tt.expectError {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tt.errorContains)
				} else {
					require.NoError(t, err)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// TICKET NUMBER GENERATION TEST
// ========================================

func TestGenerateTicketNumber(t *testing.T) {
	// Test that ticket numbers are generated in expected format
	numbers := make(map[string]bool)

	for i := 0; i < 100; i++ {
		num := generateTicketNumber()
		assert.Regexp(t, `^TKT-\d{6}$`, num, "Ticket number should match format TKT-XXXXXX")
		numbers[num] = true
	}

	// All generated numbers should be unique (with extremely high probability)
	assert.GreaterOrEqual(t, len(numbers), 95, "At least 95% of ticket numbers should be unique")
}

// ========================================
// CONSTANTS TESTS
// ========================================

func TestTicketStatus_Constants(t *testing.T) {
	assert.Equal(t, TicketStatus("open"), TicketStatusOpen)
	assert.Equal(t, TicketStatus("in_progress"), TicketStatusInProgress)
	assert.Equal(t, TicketStatus("waiting_on_user"), TicketStatusWaiting)
	assert.Equal(t, TicketStatus("resolved"), TicketStatusResolved)
	assert.Equal(t, TicketStatus("closed"), TicketStatusClosed)
	assert.Equal(t, TicketStatus("escalated"), TicketStatusEscalated)
}

func TestTicketPriority_Constants(t *testing.T) {
	assert.Equal(t, TicketPriority("low"), PriorityLow)
	assert.Equal(t, TicketPriority("medium"), PriorityMedium)
	assert.Equal(t, TicketPriority("high"), PriorityHigh)
	assert.Equal(t, TicketPriority("urgent"), PriorityUrgent)
}

func TestTicketCategory_Constants(t *testing.T) {
	assert.Equal(t, TicketCategory("payment"), CategoryPayment)
	assert.Equal(t, TicketCategory("ride"), CategoryRide)
	assert.Equal(t, TicketCategory("driver"), CategoryDriver)
	assert.Equal(t, TicketCategory("account"), CategoryAccount)
	assert.Equal(t, TicketCategory("safety"), CategorySafety)
	assert.Equal(t, TicketCategory("promo"), CategoryPromo)
	assert.Equal(t, TicketCategory("app_issue"), CategoryApp)
	assert.Equal(t, TicketCategory("feedback"), CategoryFeedback)
	assert.Equal(t, TicketCategory("lost_item"), CategoryLostItem)
	assert.Equal(t, TicketCategory("fare_dispute"), CategoryFareDispute)
	assert.Equal(t, TicketCategory("other"), CategoryOther)
}

func TestMessageSender_Constants(t *testing.T) {
	assert.Equal(t, MessageSender("user"), SenderUser)
	assert.Equal(t, MessageSender("agent"), SenderAgent)
	assert.Equal(t, MessageSender("system"), SenderSystem)
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func ptrPriority(p TicketPriority) *TicketPriority {
	return &p
}
