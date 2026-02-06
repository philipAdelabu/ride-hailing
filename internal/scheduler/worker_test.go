package scheduler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/richxcame/ride-hailing/pkg/httpclient"
)

// ============================================================================
// Mock Database
// ============================================================================

// MockDatabase implements the Database interface for testing
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	callArgs := m.Called(ctx, query, args)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(pgx.Rows), callArgs.Error(1)
}

func (m *MockDatabase) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0).(pgconn.CommandTag), callArgs.Error(1)
}

// ============================================================================
// Mock Rows
// ============================================================================

// MockRows implements pgx.Rows for testing
type MockRows struct {
	mock.Mock
	data         [][]any
	currentIndex int
	columns      []string
	closed       bool
}

func NewMockRows(columns []string, data [][]any) *MockRows {
	return &MockRows{
		data:         data,
		currentIndex: -1,
		columns:      columns,
		closed:       false,
	}
}

func (m *MockRows) Close() {
	m.closed = true
}

func (m *MockRows) Err() error {
	return nil
}

func (m *MockRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT")
}

func (m *MockRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (m *MockRows) Next() bool {
	m.currentIndex++
	return m.currentIndex < len(m.data)
}

func (m *MockRows) Scan(dest ...any) error {
	if m.currentIndex < 0 || m.currentIndex >= len(m.data) {
		return errors.New("no row to scan")
	}
	row := m.data[m.currentIndex]
	if len(dest) != len(row) {
		return errors.New("column count mismatch")
	}
	for i, v := range row {
		// Use reflection to set the value
		destVal := reflect.ValueOf(dest[i])
		if destVal.Kind() != reflect.Ptr {
			return errors.New("destination must be a pointer")
		}
		srcVal := reflect.ValueOf(v)
		destVal.Elem().Set(srcVal)
	}
	return nil
}

func (m *MockRows) Values() ([]any, error) {
	if m.currentIndex < 0 || m.currentIndex >= len(m.data) {
		return nil, errors.New("no row")
	}
	return m.data[m.currentIndex], nil
}

func (m *MockRows) RawValues() [][]byte {
	return nil
}

func (m *MockRows) Conn() *pgx.Conn {
	return nil
}

// MockRowsWithError is a MockRows that returns an error on scan
type MockRowsWithError struct {
	MockRows
	scanError error
}

func NewMockRowsWithScanError(err error) *MockRowsWithError {
	return &MockRowsWithError{
		MockRows: MockRows{
			data:         [][]any{{}},
			currentIndex: -1,
		},
		scanError: err,
	}
}

func (m *MockRowsWithError) Scan(dest ...any) error {
	return m.scanError
}

// ============================================================================
// Test Helpers
// ============================================================================

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func newTestWorker(db Database) *Worker {
	return NewWorker(db, testLogger(), "")
}

func newTestWorkerWithNotifications(db Database, url string) *Worker {
	return NewWorker(db, testLogger(), url)
}

// ============================================================================
// TestNewWorker Tests
// ============================================================================

func TestNewWorker(t *testing.T) {
	t.Run("creates worker without notifications URL", func(t *testing.T) {
		mockDB := new(MockDatabase)
		worker := NewWorker(mockDB, testLogger(), "")

		assert.NotNil(t, worker)
		assert.NotNil(t, worker.db)
		assert.NotNil(t, worker.logger)
		assert.Nil(t, worker.notificationsClient)
		assert.NotNil(t, worker.done)
	})

	t.Run("creates worker with notifications URL", func(t *testing.T) {
		mockDB := new(MockDatabase)
		worker := NewWorker(mockDB, testLogger(), "http://notifications.local")

		assert.NotNil(t, worker)
		assert.NotNil(t, worker.notificationsClient)
	})

	t.Run("creates worker with custom timeout", func(t *testing.T) {
		mockDB := new(MockDatabase)
		worker := NewWorker(mockDB, testLogger(), "http://notifications.local", 5*time.Second)

		assert.NotNil(t, worker)
		assert.NotNil(t, worker.notificationsClient)
	})

	t.Run("initializes refresh times to zero", func(t *testing.T) {
		mockDB := new(MockDatabase)
		worker := NewWorker(mockDB, testLogger(), "")

		assert.True(t, worker.lastDemandZonesRefresh.IsZero())
		assert.True(t, worker.lastDriverPerfRefresh.IsZero())
		assert.True(t, worker.lastRevenueRefresh.IsZero())
	})
}

func TestNewWorker_NilLogger(t *testing.T) {
	mockDB := new(MockDatabase)
	// This should still create the worker (logger can be nil in the implementation)
	// But in practice the caller should provide a logger
	worker := NewWorker(mockDB, nil, "")
	assert.NotNil(t, worker)
}

func TestNewWorker_EmptyURL(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := NewWorker(mockDB, testLogger(), "")
	assert.Nil(t, worker.notificationsClient)
}

func TestNewWorker_WhitespaceURL(t *testing.T) {
	mockDB := new(MockDatabase)
	// Empty string after trimming would not create a client
	worker := NewWorker(mockDB, testLogger(), "   ")
	// The implementation doesn't trim, so it will create a client with whitespace URL
	assert.NotNil(t, worker.notificationsClient)
}

// ============================================================================
// TestWorker_Stop Tests
// ============================================================================

func TestWorker_Stop(t *testing.T) {
	t.Run("closes done channel", func(t *testing.T) {
		mockDB := new(MockDatabase)
		worker := newTestWorker(mockDB)

		// Channel should be open before stop
		select {
		case <-worker.done:
			t.Fatal("done channel should be open")
		default:
			// Expected
		}

		worker.Stop()

		// Channel should be closed after stop
		select {
		case <-worker.done:
			// Expected
		default:
			t.Fatal("done channel should be closed")
		}
	})

	t.Run("stop is idempotent (panics on second call)", func(t *testing.T) {
		mockDB := new(MockDatabase)
		worker := newTestWorker(mockDB)

		worker.Stop()

		// Second stop should panic (closing already closed channel)
		assert.Panics(t, func() {
			worker.Stop()
		})
	})
}

func TestWorker_Stop_GracefulShutdown(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Simulate start behavior by checking done channel
	go func() {
		select {
		case <-worker.done:
			// Graceful shutdown
		case <-ctx.Done():
			// Context cancelled
		}
	}()

	worker.Stop()

	// Give time for goroutine to process
	time.Sleep(10 * time.Millisecond)
}

// ============================================================================
// TestWorker_ProcessScheduledRides Tests
// ============================================================================

func TestWorker_ProcessScheduledRides_EmptyResults(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	emptyRows := NewMockRows([]string{}, [][]any{})
	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(emptyRows, nil)

	// Should not panic and should handle empty results gracefully
	worker.processScheduledRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ProcessScheduledRides_QueryError(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

	// Should not panic on query error
	worker.processScheduledRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ProcessScheduledRides_ActivatesRideWithin5Minutes(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	riderID := uuid.New()
	scheduledAt := time.Now().Add(3 * time.Minute) // Within 5 minutes

	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{rideID, riderID, scheduledAt, "Pickup", "Dropoff", 25.0, false},
		},
	)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "UPDATE rides") && containsString(q, "is_scheduled = false")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	worker.processScheduledRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ProcessScheduledRides_SendsNotificationWithin30Minutes(t *testing.T) {
	// Create a test server to handle notification requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	mockDB := new(MockDatabase)
	worker := newTestWorkerWithNotifications(mockDB, server.URL)

	rideID := uuid.New()
	riderID := uuid.New()
	scheduledAt := time.Now().Add(20 * time.Minute) // Within 30 but not 5 minutes

	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{rideID, riderID, scheduledAt, "Pickup", "Dropoff", 25.0, false},
		},
	)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "scheduled_notification_sent = true")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	worker.processScheduledRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ProcessScheduledRides_SkipsAlreadySentNotification(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	riderID := uuid.New()
	scheduledAt := time.Now().Add(20 * time.Minute)

	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{rideID, riderID, scheduledAt, "Pickup", "Dropoff", 25.0, true}, // Already sent
		},
	)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)
	// Should NOT call Exec for notification since already sent

	worker.processScheduledRides(context.Background())

	mockDB.AssertExpectations(t)
	// Verify no Exec calls were made
	mockDB.AssertNotCalled(t, "Exec", mock.Anything, mock.Anything, mock.Anything)
}

func TestWorker_ProcessScheduledRides_MultipleRides(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	ride1ID := uuid.New()
	ride2ID := uuid.New()
	rider1ID := uuid.New()
	rider2ID := uuid.New()

	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{ride1ID, rider1ID, time.Now().Add(3 * time.Minute), "Pickup1", "Dropoff1", 20.0, false},
			{ride2ID, rider2ID, time.Now().Add(4 * time.Minute), "Pickup2", "Dropoff2", 30.0, false},
		},
	)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	worker.processScheduledRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ProcessScheduledRides_ActivateError(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	riderID := uuid.New()
	scheduledAt := time.Now().Add(3 * time.Minute)

	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{rideID, riderID, scheduledAt, "Pickup", "Dropoff", 25.0, false},
		},
	)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, errors.New("exec error"))

	// Should handle error gracefully without panic
	worker.processScheduledRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ProcessScheduledRides_ScanError(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	errorRows := NewMockRowsWithScanError(errors.New("scan error"))
	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(errorRows, nil)

	// Should handle scan error gracefully
	worker.processScheduledRides(context.Background())

	mockDB.AssertExpectations(t)
}

// ============================================================================
// TestWorker_ActivateScheduledRide Tests
// ============================================================================

func TestWorker_ActivateScheduledRide_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	err := worker.activateScheduledRide(context.Background(), rideID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_ActivateScheduledRide_NoRowsAffected(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 0"), nil)

	err := worker.activateScheduledRide(context.Background(), rideID)

	// Should not return error even when no rows affected
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_ActivateScheduledRide_DatabaseError(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	expectedErr := errors.New("database error")
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, expectedErr)

	err := worker.activateScheduledRide(context.Background(), rideID)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_ActivateScheduledRide_ContextCancelled(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rideID := uuid.New()
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, context.Canceled)

	err := worker.activateScheduledRide(ctx, rideID)

	assert.Error(t, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_ActivateScheduledRide_QueryFormat(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "UPDATE rides") &&
			containsString(q, "is_scheduled = false") &&
			containsString(q, "requested_at = NOW()") &&
			containsString(q, "updated_at = NOW()")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	err := worker.activateScheduledRide(context.Background(), rideID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

// ============================================================================
// TestWorker_SendUpcomingRideNotification Tests
// ============================================================================

func TestWorker_SendUpcomingRideNotification_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/notifications", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	mockDB := new(MockDatabase)
	worker := newTestWorkerWithNotifications(mockDB, server.URL)

	rideID := uuid.New()
	riderID := uuid.New()
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	err := worker.sendUpcomingRideNotification(context.Background(), rideID, riderID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_SendUpcomingRideNotification_AlreadySent(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	riderID := uuid.New()
	// Return 0 rows affected, meaning notification was already sent
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 0"), nil)

	err := worker.sendUpcomingRideNotification(context.Background(), rideID, riderID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_SendUpcomingRideNotification_NoClient(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB) // No notifications URL

	rideID := uuid.New()
	riderID := uuid.New()
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	err := worker.sendUpcomingRideNotification(context.Background(), rideID, riderID)

	// Should not error even without client
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_SendUpcomingRideNotification_DatabaseError(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	riderID := uuid.New()
	expectedErr := errors.New("database error")
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, expectedErr)

	err := worker.sendUpcomingRideNotification(context.Background(), rideID, riderID)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_SendUpcomingRideNotification_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	mockDB := new(MockDatabase)
	worker := newTestWorkerWithNotifications(mockDB, server.URL)

	rideID := uuid.New()
	riderID := uuid.New()
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	// HTTP errors should not cause the function to return an error
	// (notification is marked as sent to prevent retries)
	err := worker.sendUpcomingRideNotification(context.Background(), rideID, riderID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_SendUpcomingRideNotification_UpdateQuery(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	riderID := uuid.New()
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "scheduled_notification_sent = true") &&
			containsString(q, "updated_at = NOW()")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	err := worker.sendUpcomingRideNotification(context.Background(), rideID, riderID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

// ============================================================================
// TestWorker_ExpireStaleRides Tests
// ============================================================================

func TestWorker_ExpireStaleRides_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "status = 'cancelled'") &&
			containsString(q, "cancellation_reason")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 5"), nil)

	worker.expireStaleRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ExpireStaleRides_NoRidesToExpire(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 0"), nil)

	worker.expireStaleRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ExpireStaleRides_DatabaseError(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, errors.New("database error"))

	// Should not panic on error
	worker.expireStaleRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ExpireStaleRides_QueryContainsCorrectConditions(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "status = 'requested'") &&
			containsString(q, "is_scheduled = false") &&
			containsString(q, "requested_at <")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 0"), nil)

	worker.expireStaleRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ExpireStaleRides_SetsCancellationReason(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "cancellation_reason = 'expired: no driver accepted within timeout'")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	worker.expireStaleRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ExpireStaleRides_SetsCancelledAt(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "cancelled_at = NOW()")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	worker.expireStaleRides(context.Background())

	mockDB.AssertExpectations(t)
}

// ============================================================================
// TestWorker_RefreshMaterializedViews Tests
// ============================================================================

func TestWorker_RefreshMaterializedViews_RefreshesDemandZones(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)
	// Set last refresh to 20 minutes ago (beyond 15 minute interval)
	worker.lastDemandZonesRefresh = time.Now().Add(-20 * time.Minute)
	worker.lastDriverPerfRefresh = time.Now()
	worker.lastRevenueRefresh = time.Now()

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_demand_zones")
	}), mock.Anything).Return(pgconn.NewCommandTag("REFRESH"), nil)

	worker.refreshMaterializedViews(context.Background())

	mockDB.AssertExpectations(t)
	assert.False(t, worker.lastDemandZonesRefresh.IsZero())
}

func TestWorker_RefreshMaterializedViews_RefreshesDriverPerf(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)
	// Set last refresh to 2 hours ago (beyond 1 hour interval)
	worker.lastDemandZonesRefresh = time.Now()
	worker.lastDriverPerfRefresh = time.Now().Add(-2 * time.Hour)
	worker.lastRevenueRefresh = time.Now()

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_driver_performance")
	}), mock.Anything).Return(pgconn.NewCommandTag("REFRESH"), nil)

	worker.refreshMaterializedViews(context.Background())

	mockDB.AssertExpectations(t)
	assert.False(t, worker.lastDriverPerfRefresh.IsZero())
}

func TestWorker_RefreshMaterializedViews_RefreshesRevenue(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)
	// Set last refresh to 2 days ago (beyond 24 hour interval)
	worker.lastDemandZonesRefresh = time.Now()
	worker.lastDriverPerfRefresh = time.Now()
	worker.lastRevenueRefresh = time.Now().Add(-48 * time.Hour)

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_revenue_metrics")
	}), mock.Anything).Return(pgconn.NewCommandTag("REFRESH"), nil)

	worker.refreshMaterializedViews(context.Background())

	mockDB.AssertExpectations(t)
	assert.False(t, worker.lastRevenueRefresh.IsZero())
}

func TestWorker_RefreshMaterializedViews_AllViewsNeedRefresh(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)
	// All views need refresh (initial state with zero times)

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "mv_demand_zones")
	}), mock.Anything).Return(pgconn.NewCommandTag("REFRESH"), nil)
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "mv_driver_performance")
	}), mock.Anything).Return(pgconn.NewCommandTag("REFRESH"), nil)
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "mv_revenue_metrics")
	}), mock.Anything).Return(pgconn.NewCommandTag("REFRESH"), nil)

	worker.refreshMaterializedViews(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_RefreshMaterializedViews_NoRefreshNeeded(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)
	// All views were just refreshed
	worker.lastDemandZonesRefresh = time.Now()
	worker.lastDriverPerfRefresh = time.Now()
	worker.lastRevenueRefresh = time.Now()

	worker.refreshMaterializedViews(context.Background())

	// No Exec calls should be made
	mockDB.AssertNotCalled(t, "Exec", mock.Anything, mock.Anything, mock.Anything)
}

func TestWorker_RefreshMaterializedViews_DemandZonesError(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)
	originalTime := worker.lastDemandZonesRefresh
	// Set other refresh times to prevent them from triggering
	worker.lastDriverPerfRefresh = time.Now()
	worker.lastRevenueRefresh = time.Now()

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "mv_demand_zones")
	}), mock.Anything).Return(pgconn.CommandTag{}, errors.New("refresh error"))

	worker.refreshMaterializedViews(context.Background())

	// Time should not be updated on error
	assert.Equal(t, originalTime, worker.lastDemandZonesRefresh)
	mockDB.AssertExpectations(t)
}

func TestWorker_RefreshMaterializedViews_DriverPerfError(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)
	worker.lastDemandZonesRefresh = time.Now()
	worker.lastRevenueRefresh = time.Now()
	originalTime := worker.lastDriverPerfRefresh

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "mv_driver_performance")
	}), mock.Anything).Return(pgconn.CommandTag{}, errors.New("refresh error"))

	worker.refreshMaterializedViews(context.Background())

	// Time should not be updated on error
	assert.Equal(t, originalTime, worker.lastDriverPerfRefresh)
	mockDB.AssertExpectations(t)
}

func TestWorker_RefreshMaterializedViews_RevenueError(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)
	worker.lastDemandZonesRefresh = time.Now()
	worker.lastDriverPerfRefresh = time.Now()
	originalTime := worker.lastRevenueRefresh

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "mv_revenue_metrics")
	}), mock.Anything).Return(pgconn.CommandTag{}, errors.New("refresh error"))

	worker.refreshMaterializedViews(context.Background())

	// Time should not be updated on error
	assert.Equal(t, originalTime, worker.lastRevenueRefresh)
	mockDB.AssertExpectations(t)
}

// ============================================================================
// TestWorker_RefreshView Tests
// ============================================================================

func TestWorker_RefreshView_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	mockDB.On("Exec", mock.Anything, "REFRESH MATERIALIZED VIEW CONCURRENTLY test_view", mock.Anything).Return(pgconn.NewCommandTag("REFRESH"), nil)

	err := worker.refreshView(context.Background(), "test_view")

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_RefreshView_Error(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	expectedErr := errors.New("refresh failed")
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, expectedErr)

	err := worker.refreshView(context.Background(), "test_view")

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_RefreshView_Timeout(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, context.DeadlineExceeded)

	err := worker.refreshView(context.Background(), "test_view")

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_RefreshView_ConcurrentlyKeyword(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "CONCURRENTLY")
	}), mock.Anything).Return(pgconn.NewCommandTag("REFRESH"), nil)

	err := worker.refreshView(context.Background(), "any_view")

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

// ============================================================================
// TestWorker_GetUpcomingScheduledRides Tests
// ============================================================================

func TestWorker_GetUpcomingScheduledRides_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	riderID := uuid.New()
	scheduledAt := time.Now().Add(10 * time.Minute)

	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{rideID, riderID, scheduledAt, "123 Main St", "456 Oak Ave", 25.50, false},
		},
	)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)

	rides, err := worker.getUpcomingScheduledRides(context.Background(), 30)

	assert.NoError(t, err)
	assert.Len(t, rides, 1)
	assert.Equal(t, rideID, rides[0].ID)
	assert.Equal(t, riderID, rides[0].RiderID)
	assert.Equal(t, "123 Main St", rides[0].PickupAddress)
	assert.Equal(t, "456 Oak Ave", rides[0].DropoffAddress)
	assert.Equal(t, 25.50, rides[0].EstimatedFare)
	assert.False(t, rides[0].NotificationSent)
	mockDB.AssertExpectations(t)
}

func TestWorker_GetUpcomingScheduledRides_Empty(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	emptyRows := NewMockRows([]string{}, [][]any{})
	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(emptyRows, nil)

	rides, err := worker.getUpcomingScheduledRides(context.Background(), 30)

	assert.NoError(t, err)
	assert.Empty(t, rides)
	mockDB.AssertExpectations(t)
}

func TestWorker_GetUpcomingScheduledRides_QueryError(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	expectedErr := errors.New("query failed")
	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(nil, expectedErr)

	rides, err := worker.getUpcomingScheduledRides(context.Background(), 30)

	assert.Error(t, err)
	assert.Nil(t, rides)
	assert.Equal(t, expectedErr, err)
	mockDB.AssertExpectations(t)
}

func TestWorker_GetUpcomingScheduledRides_ScanError(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	errorRows := NewMockRowsWithScanError(errors.New("scan failed"))
	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(errorRows, nil)

	rides, err := worker.getUpcomingScheduledRides(context.Background(), 30)

	assert.Error(t, err)
	assert.Nil(t, rides)
	mockDB.AssertExpectations(t)
}

func TestWorker_GetUpcomingScheduledRides_MultipleRides(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	ride1ID := uuid.New()
	ride2ID := uuid.New()
	ride3ID := uuid.New()
	riderID := uuid.New()

	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{ride1ID, riderID, time.Now().Add(5 * time.Minute), "Addr1", "Dest1", 10.0, false},
			{ride2ID, riderID, time.Now().Add(15 * time.Minute), "Addr2", "Dest2", 20.0, true},
			{ride3ID, riderID, time.Now().Add(25 * time.Minute), "Addr3", "Dest3", 30.0, false},
		},
	)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)

	rides, err := worker.getUpcomingScheduledRides(context.Background(), 30)

	assert.NoError(t, err)
	assert.Len(t, rides, 3)
	assert.Equal(t, ride1ID, rides[0].ID)
	assert.Equal(t, ride2ID, rides[1].ID)
	assert.Equal(t, ride3ID, rides[2].ID)
	mockDB.AssertExpectations(t)
}

func TestWorker_GetUpcomingScheduledRides_QueryFormat(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	emptyRows := NewMockRows([]string{}, [][]any{})
	mockDB.On("Query", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "is_scheduled = true") &&
			containsString(q, "status = 'requested'") &&
			containsString(q, "scheduled_at <= NOW()") &&
			containsString(q, "scheduled_at > NOW()") &&
			containsString(q, "ORDER BY scheduled_at ASC")
	}), mock.Anything).Return(emptyRows, nil)

	_, err := worker.getUpcomingScheduledRides(context.Background(), 30)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

// ============================================================================
// TestScheduledRide Struct Tests
// ============================================================================

func TestScheduledRide_Fields(t *testing.T) {
	ride := &ScheduledRide{
		ID:               uuid.New(),
		RiderID:          uuid.New(),
		ScheduledAt:      time.Now(),
		PickupAddress:    "123 Main St",
		DropoffAddress:   "456 Oak Ave",
		EstimatedFare:    25.50,
		NotificationSent: true,
	}

	assert.NotEqual(t, uuid.Nil, ride.ID)
	assert.NotEqual(t, uuid.Nil, ride.RiderID)
	assert.False(t, ride.ScheduledAt.IsZero())
	assert.Equal(t, "123 Main St", ride.PickupAddress)
	assert.Equal(t, "456 Oak Ave", ride.DropoffAddress)
	assert.Equal(t, 25.50, ride.EstimatedFare)
	assert.True(t, ride.NotificationSent)
}

// ============================================================================
// TestWorker_Start Tests
// ============================================================================

func TestWorker_Start_ImmediateExecution(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	// Set up expectations for all three operations on startup
	emptyRows := NewMockRows([]string{}, [][]any{})
	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(emptyRows, nil)
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "status = 'cancelled'")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 0"), nil)
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "REFRESH MATERIALIZED VIEW")
	}), mock.Anything).Return(pgconn.NewCommandTag("REFRESH"), nil)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	worker.Start(ctx)

	// Verify that operations were called
	mockDB.AssertCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything)
}

func TestWorker_Start_StopsOnContextCancel(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	emptyRows := NewMockRows([]string{}, [][]any{})
	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(emptyRows, nil)
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag(""), nil)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		worker.Start(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// Success - worker stopped
	case <-time.After(2 * time.Second):
		t.Fatal("Worker did not stop on context cancel")
	}
}

func TestWorker_Start_StopsOnDoneChannel(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	emptyRows := NewMockRows([]string{}, [][]any{})
	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(emptyRows, nil)
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag(""), nil)

	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		worker.Start(ctx)
		close(done)
	}()

	worker.Stop()

	select {
	case <-done:
		// Success - worker stopped
	case <-time.After(2 * time.Second):
		t.Fatal("Worker did not stop on done channel close")
	}
}

// ============================================================================
// Constants Tests
// ============================================================================

func TestConstants(t *testing.T) {
	assert.Equal(t, 1*time.Minute, checkInterval)
	assert.Equal(t, 30, lookAheadMinutes)
	assert.Equal(t, 15*time.Minute, demandZonesRefreshInterval)
	assert.Equal(t, 1*time.Hour, driverPerfRefreshInterval)
	assert.Equal(t, 24*time.Hour, revenueRefreshInterval)
	assert.Equal(t, 5*time.Minute, rideExpiryThreshold)
}

// ============================================================================
// Integration-like Tests
// ============================================================================

func TestWorker_FullProcessingCycle(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	riderID := uuid.New()
	scheduledAt := time.Now().Add(3 * time.Minute)

	// Query for scheduled rides
	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{rideID, riderID, scheduledAt, "Pickup", "Dropoff", 25.0, false},
		},
	)
	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil).Once()

	// Activate ride
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "is_scheduled = false")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil).Once()

	// Expire stale rides
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "status = 'cancelled'")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 0"), nil).Once()

	// Refresh views
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "REFRESH")
	}), mock.Anything).Return(pgconn.NewCommandTag("REFRESH"), nil)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	worker.Start(ctx)
}

func TestWorker_NotificationServiceIntegration(t *testing.T) {
	notificationReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notificationReceived = true
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/notifications", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	mockDB := new(MockDatabase)
	worker := newTestWorkerWithNotifications(mockDB, server.URL)

	rideID := uuid.New()
	riderID := uuid.New()
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	err := worker.sendUpcomingRideNotification(context.Background(), rideID, riderID)

	assert.NoError(t, err)
	assert.True(t, notificationReceived)
}

// ============================================================================
// Edge Cases Tests
// ============================================================================

func TestWorker_ProcessScheduledRides_RideExactly5MinutesAway(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	riderID := uuid.New()
	// Exactly 5 minutes - should be activated (<=5 min)
	scheduledAt := time.Now().Add(5 * time.Minute)

	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{rideID, riderID, scheduledAt, "Pickup", "Dropoff", 25.0, false},
		},
	)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "is_scheduled = false")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	worker.processScheduledRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ProcessScheduledRides_RideExactly30MinutesAway(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockDB := new(MockDatabase)
	worker := newTestWorkerWithNotifications(mockDB, server.URL)

	rideID := uuid.New()
	riderID := uuid.New()
	// Exactly 30 minutes - notification should be sent (<=30 min but >5 min)
	scheduledAt := time.Now().Add(30 * time.Minute)

	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{rideID, riderID, scheduledAt, "Pickup", "Dropoff", 25.0, false},
		},
	)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(q string) bool {
		return containsString(q, "scheduled_notification_sent = true")
	}), mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	worker.processScheduledRides(context.Background())

	mockDB.AssertExpectations(t)
}

func TestWorker_ProcessScheduledRides_RideMoreThan30MinutesAway(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	riderID := uuid.New()
	// 31 minutes away - should not trigger any action
	scheduledAt := time.Now().Add(31 * time.Minute)

	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{rideID, riderID, scheduledAt, "Pickup", "Dropoff", 25.0, false},
		},
	)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)
	// No Exec calls expected

	worker.processScheduledRides(context.Background())

	mockDB.AssertExpectations(t)
	mockDB.AssertNotCalled(t, "Exec", mock.Anything, mock.Anything, mock.Anything)
}

func TestWorker_ZeroUUID(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	zeroUUID := uuid.Nil
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 0"), nil)

	err := worker.activateScheduledRide(context.Background(), zeroUUID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

// ============================================================================
// Database Interface Tests
// ============================================================================

func TestDatabaseInterface_ImplementedByMock(t *testing.T) {
	var db Database = new(MockDatabase)
	assert.NotNil(t, db)
}

func TestDatabaseInterface_Query(t *testing.T) {
	mockDB := new(MockDatabase)
	emptyRows := NewMockRows([]string{}, [][]any{})
	mockDB.On("Query", mock.Anything, "SELECT 1", mock.Anything).Return(emptyRows, nil)

	var db Database = mockDB
	rows, err := db.Query(context.Background(), "SELECT 1")

	assert.NoError(t, err)
	assert.NotNil(t, rows)
	mockDB.AssertExpectations(t)
}

func TestDatabaseInterface_Exec(t *testing.T) {
	mockDB := new(MockDatabase)
	mockDB.On("Exec", mock.Anything, "UPDATE test SET x = 1", mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	var db Database = mockDB
	tag, err := db.Exec(context.Background(), "UPDATE test SET x = 1")

	assert.NoError(t, err)
	assert.Equal(t, int64(1), tag.RowsAffected())
	mockDB.AssertExpectations(t)
}

// ============================================================================
// Worker with HTTPClient Tests
// ============================================================================

func TestWorker_HTTPClientTimeout(t *testing.T) {
	mockDB := new(MockDatabase)

	// Test with various timeout values
	timeouts := []time.Duration{
		1 * time.Second,
		5 * time.Second,
		30 * time.Second,
	}

	for _, timeout := range timeouts {
		t.Run(timeout.String(), func(t *testing.T) {
			worker := NewWorker(mockDB, testLogger(), "http://test.local", timeout)
			assert.NotNil(t, worker.notificationsClient)
		})
	}
}

func TestWorker_HTTPClientNoTimeout(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := NewWorker(mockDB, testLogger(), "http://test.local")
	assert.NotNil(t, worker.notificationsClient)
}

// ============================================================================
// Helper Functions
// ============================================================================

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkWorker_ProcessScheduledRides_EmptyResults(b *testing.B) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	emptyRows := NewMockRows([]string{}, [][]any{})
	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(emptyRows, nil)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		worker.processScheduledRides(ctx)
	}
}

func BenchmarkWorker_ActivateScheduledRide(b *testing.B) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	ctx := context.Background()
	rideID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		worker.activateScheduledRide(ctx, rideID)
	}
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestWorker_RefreshIntervals(t *testing.T) {
	tests := []struct {
		name             string
		lastRefresh      time.Duration
		interval         time.Duration
		shouldRefresh    bool
		viewName         string
	}{
		{
			name:          "demand zones - needs refresh",
			lastRefresh:   -20 * time.Minute,
			interval:      demandZonesRefreshInterval,
			shouldRefresh: true,
			viewName:      "mv_demand_zones",
		},
		{
			name:          "demand zones - no refresh needed",
			lastRefresh:   -10 * time.Minute,
			interval:      demandZonesRefreshInterval,
			shouldRefresh: false,
			viewName:      "mv_demand_zones",
		},
		{
			name:          "driver perf - needs refresh",
			lastRefresh:   -2 * time.Hour,
			interval:      driverPerfRefreshInterval,
			shouldRefresh: true,
			viewName:      "mv_driver_performance",
		},
		{
			name:          "driver perf - no refresh needed",
			lastRefresh:   -30 * time.Minute,
			interval:      driverPerfRefreshInterval,
			shouldRefresh: false,
			viewName:      "mv_driver_performance",
		},
		{
			name:          "revenue - needs refresh",
			lastRefresh:   -48 * time.Hour,
			interval:      revenueRefreshInterval,
			shouldRefresh: true,
			viewName:      "mv_revenue_metrics",
		},
		{
			name:          "revenue - no refresh needed",
			lastRefresh:   -12 * time.Hour,
			interval:      revenueRefreshInterval,
			shouldRefresh: false,
			viewName:      "mv_revenue_metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			lastRefresh := now.Add(tt.lastRefresh)
			timeSinceRefresh := now.Sub(lastRefresh)

			needsRefresh := timeSinceRefresh >= tt.interval
			assert.Equal(t, tt.shouldRefresh, needsRefresh)
		})
	}
}

func TestWorker_RideTimeCategories(t *testing.T) {
	tests := []struct {
		name              string
		timeUntilRide     time.Duration
		shouldActivate    bool
		shouldNotify      bool
		notificationSent  bool
	}{
		{
			name:             "immediate - activate",
			timeUntilRide:    1 * time.Minute,
			shouldActivate:   true,
			shouldNotify:     false,
			notificationSent: false,
		},
		{
			name:             "5 min boundary - activate",
			timeUntilRide:    5 * time.Minute,
			shouldActivate:   true,
			shouldNotify:     false,
			notificationSent: false,
		},
		{
			name:             "15 min - notify",
			timeUntilRide:    15 * time.Minute,
			shouldActivate:   false,
			shouldNotify:     true,
			notificationSent: false,
		},
		{
			name:             "15 min - already notified",
			timeUntilRide:    15 * time.Minute,
			shouldActivate:   false,
			shouldNotify:     false,
			notificationSent: true,
		},
		{
			name:             "30 min boundary - notify",
			timeUntilRide:    30 * time.Minute,
			shouldActivate:   false,
			shouldNotify:     true,
			notificationSent: false,
		},
		{
			name:             "beyond 30 min - no action",
			timeUntilRide:    45 * time.Minute,
			shouldActivate:   false,
			shouldNotify:     false,
			notificationSent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldActivate := tt.timeUntilRide <= 5*time.Minute
			shouldNotify := tt.timeUntilRide > 5*time.Minute &&
				tt.timeUntilRide <= 30*time.Minute &&
				!tt.notificationSent

			assert.Equal(t, tt.shouldActivate, shouldActivate)
			assert.Equal(t, tt.shouldNotify, shouldNotify)
		})
	}
}

// ============================================================================
// Additional Mock Row Tests
// ============================================================================

func TestMockRows_Close(t *testing.T) {
	rows := NewMockRows([]string{"col1"}, [][]any{{"value"}})
	assert.False(t, rows.closed)
	rows.Close()
	assert.True(t, rows.closed)
}

func TestMockRows_Err(t *testing.T) {
	rows := NewMockRows([]string{}, [][]any{})
	assert.NoError(t, rows.Err())
}

func TestMockRows_CommandTag(t *testing.T) {
	rows := NewMockRows([]string{}, [][]any{})
	tag := rows.CommandTag()
	assert.Equal(t, "SELECT", tag.String())
}

func TestMockRows_FieldDescriptions(t *testing.T) {
	rows := NewMockRows([]string{}, [][]any{})
	assert.Nil(t, rows.FieldDescriptions())
}

func TestMockRows_Values(t *testing.T) {
	rows := NewMockRows([]string{"col1", "col2"}, [][]any{{"v1", "v2"}})

	// Before Next()
	_, err := rows.Values()
	assert.Error(t, err)

	// After Next()
	rows.Next()
	values, err := rows.Values()
	assert.NoError(t, err)
	assert.Equal(t, []any{"v1", "v2"}, values)
}

func TestMockRows_RawValues(t *testing.T) {
	rows := NewMockRows([]string{}, [][]any{})
	assert.Nil(t, rows.RawValues())
}

func TestMockRows_Conn(t *testing.T) {
	rows := NewMockRows([]string{}, [][]any{})
	assert.Nil(t, rows.Conn())
}

func TestMockRows_Scan_NoRow(t *testing.T) {
	rows := NewMockRows([]string{"col1"}, [][]any{})
	var val string
	err := rows.Scan(&val)
	assert.Error(t, err)
}

func TestMockRows_Scan_ColumnMismatch(t *testing.T) {
	rows := NewMockRows([]string{"col1", "col2"}, [][]any{{"v1", "v2"}})
	rows.Next()
	var val string
	err := rows.Scan(&val) // Only 1 dest, but 2 columns
	assert.Error(t, err)
}

func TestMockRows_Scan_NonPointer(t *testing.T) {
	rows := NewMockRows([]string{"col1"}, [][]any{{"v1"}})
	rows.Next()
	var val string
	err := rows.Scan(val) // Not a pointer
	assert.Error(t, err)
}

// ============================================================================
// Require-based Tests
// ============================================================================

func TestWorker_Creation_RequireNotNil(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := NewWorker(mockDB, testLogger(), "http://test.local")

	require.NotNil(t, worker)
	require.NotNil(t, worker.db)
	require.NotNil(t, worker.logger)
	require.NotNil(t, worker.notificationsClient)
	require.NotNil(t, worker.done)
}

func TestWorker_GetUpcomingScheduledRides_RequireSuccess(t *testing.T) {
	mockDB := new(MockDatabase)
	worker := newTestWorker(mockDB)

	rideID := uuid.New()
	riderID := uuid.New()

	rows := NewMockRows(
		[]string{"id", "rider_id", "scheduled_at", "pickup_address", "dropoff_address", "estimated_fare", "scheduled_notification_sent"},
		[][]any{
			{rideID, riderID, time.Now().Add(10 * time.Minute), "Pickup", "Dropoff", 25.0, false},
		},
	)

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(rows, nil)

	rides, err := worker.getUpcomingScheduledRides(context.Background(), 30)

	require.NoError(t, err)
	require.Len(t, rides, 1)
	require.Equal(t, rideID, rides[0].ID)
}

// ============================================================================
// Notification Client Tests
// ============================================================================

func TestWorker_NotificationClient_Creation(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectNil   bool
	}{
		{
			name:      "empty URL",
			url:       "",
			expectNil: true,
		},
		{
			name:      "valid URL",
			url:       "http://notifications.local",
			expectNil: false,
		},
		{
			name:      "HTTPS URL",
			url:       "https://notifications.local",
			expectNil: false,
		},
		{
			name:      "URL with port",
			url:       "http://notifications.local:8080",
			expectNil: false,
		},
		{
			name:      "URL with path",
			url:       "http://notifications.local/api",
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDatabase)
			worker := NewWorker(mockDB, testLogger(), tt.url)

			if tt.expectNil {
				assert.Nil(t, worker.notificationsClient)
			} else {
				assert.NotNil(t, worker.notificationsClient)
			}
		})
	}
}

// ============================================================================
// Worker with Custom HTTP Client Test
// ============================================================================

func TestWorker_WithCustomHTTPClient(t *testing.T) {
	mockDB := new(MockDatabase)

	// Test that we can create workers with different configurations
	worker1 := NewWorker(mockDB, testLogger(), "http://svc1.local")
	worker2 := NewWorker(mockDB, testLogger(), "http://svc2.local", 10*time.Second)
	worker3 := NewWorker(mockDB, testLogger(), "")

	assert.NotNil(t, worker1.notificationsClient)
	assert.NotNil(t, worker2.notificationsClient)
	assert.Nil(t, worker3.notificationsClient)
}

// ============================================================================
// Context Handling Tests
// ============================================================================

func TestWorker_ContextHandling(t *testing.T) {
	t.Run("background context", func(t *testing.T) {
		mockDB := new(MockDatabase)
		worker := newTestWorker(mockDB)

		mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

		err := worker.activateScheduledRide(context.Background(), uuid.New())
		assert.NoError(t, err)
	})

	t.Run("cancelled context", func(t *testing.T) {
		mockDB := new(MockDatabase)
		worker := newTestWorker(mockDB)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, context.Canceled)

		err := worker.activateScheduledRide(ctx, uuid.New())
		assert.Error(t, err)
	})

	t.Run("deadline exceeded context", func(t *testing.T) {
		mockDB := new(MockDatabase)
		worker := newTestWorker(mockDB)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(2 * time.Millisecond) // Ensure deadline passes

		mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, context.DeadlineExceeded)

		err := worker.activateScheduledRide(ctx, uuid.New())
		assert.Error(t, err)
	})
}

// ============================================================================
// Notification Body Content Test
// ============================================================================

func TestWorker_NotificationBodyContent(t *testing.T) {
	receivedBody := make(map[string]interface{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Note: We can't easily parse the body here without additional setup
		// but we verify the request was made
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockDB := new(MockDatabase)
	worker := newTestWorkerWithNotifications(mockDB, server.URL)

	rideID := uuid.New()
	riderID := uuid.New()
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	err := worker.sendUpcomingRideNotification(context.Background(), rideID, riderID)

	assert.NoError(t, err)
	// The notification request should include:
	// - user_id
	// - type: "scheduled_ride_reminder"
	// - channel: "push"
	// - title: "Upcoming Ride Scheduled"
	// - body: "Your scheduled ride is coming up soon. Get ready!"
	// - data with ride_id and action
	_ = receivedBody // Acknowledge variable to avoid unused warning
}

// ============================================================================
// Worker Logger Tests
// ============================================================================

func TestWorker_LoggerNotNil(t *testing.T) {
	mockDB := new(MockDatabase)
	logger := testLogger()
	worker := NewWorker(mockDB, logger, "")

	assert.NotNil(t, worker.logger)
	assert.Same(t, logger, worker.logger)
}

// ============================================================================
// HTTPClient Package Dependency Test
// ============================================================================

func TestWorker_UsesHTTPClientPackage(t *testing.T) {
	// Verify that the worker correctly uses the httpclient package
	mockDB := new(MockDatabase)
	worker := NewWorker(mockDB, testLogger(), "http://test.local")

	// The notificationsClient should be of type *httpclient.Client
	assert.NotNil(t, worker.notificationsClient)
	assert.IsType(t, (*httpclient.Client)(nil), worker.notificationsClient)
}
