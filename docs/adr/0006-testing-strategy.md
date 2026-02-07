# ADR-0006: Testing Strategy

## Status

Accepted

## Context

A ride-hailing platform handles critical operations: payments, user safety, and real-time coordination. Testing requirements:

1. **Reliability**: Payment and ride booking paths must have comprehensive coverage.
2. **Speed**: Tests must complete quickly to support CI/CD pipelines.
3. **Isolation**: Unit tests must not require external dependencies.
4. **Maintainability**: Test code must be readable and easy to update.
5. **Coverage**: Critical paths need near-complete coverage.

### Testing Pyramid

```
                    /\
                   /  \
                  / E2E\           ~10 tests
                 /------\
                /  Integ \         ~50 tests
               /----------\
              /   Unit     \       ~2600+ tests
             /--------------\
```

## Decision

Adopt **table-driven tests** with **testify/mock** for comprehensive, maintainable test coverage.

### Testing Stack

| Component | Purpose |
|-----------|---------|
| `testing` (stdlib) | Test runner |
| `testify/assert` | Assertions |
| `testify/mock` | Mock generation |
| `testify/require` | Fatal assertions |
| `test/helpers` | Shared fixtures and utilities |
| `test/mocks` | Hand-written mock implementations |

### Table-Driven Test Pattern

```go
// From internal/auth/service_test.go
func TestService_Login_Success(t *testing.T) {
    mockRepo := new(mocks.MockAuthRepository)
    service := newTestService(t, mockRepo)
    ctx := context.Background()
    req := helpers.CreateTestLoginRequest()
    testUser := helpers.CreateTestUser()

    // Mock expectations
    mockRepo.On("GetUserByEmail", mock.Anything, req.Email).Return(testUser, nil)

    // Execute
    response, err := service.Login(ctx, req)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.NotEmpty(t, response.Token)
    helpers.AssertValidJWT(t, response.Token)
    mockRepo.AssertExpectations(t)
}

func TestService_Login_UserNotFound(t *testing.T) {
    mockRepo := new(mocks.MockAuthRepository)
    service := newTestService(t, mockRepo)
    ctx := context.Background()
    req := helpers.CreateTestLoginRequest()

    mockRepo.On("GetUserByEmail", mock.Anything, req.Email).Return(nil, errors.New("not found"))

    response, err := service.Login(ctx, req)

    assert.Error(t, err)
    assert.Nil(t, response)
    var appErr *common.AppError
    assert.True(t, errors.As(err, &appErr))
    assert.Equal(t, 401, appErr.Code)
    mockRepo.AssertExpectations(t)
}
```

### Test Helpers

```go
// From test/helpers/fixtures.go
func CreateTestUser() *models.User {
    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("TestPassword123!"), bcrypt.DefaultCost)
    return &models.User{
        ID:           uuid.New(),
        Email:        "test@example.com",
        PasswordHash: string(hashedPassword),
        FirstName:    "Test",
        LastName:     "User",
        Role:         "rider",
        IsActive:     true,
        IsVerified:   true,
        CreatedAt:    time.Now(),
    }
}

func CreateTestRegisterRequest() *models.RegisterRequest {
    return &models.RegisterRequest{
        Email:       "newuser@example.com",
        Password:    "SecurePassword123!",
        PhoneNumber: "+1234567890",
        FirstName:   "New",
        LastName:    "User",
        Role:        "rider",
    }
}
```

### Mock Repository Pattern

```go
// From test/mocks/repository.go
type MockAuthRepository struct {
    mock.Mock
}

func (m *MockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
    args := m.Called(ctx, email)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthRepository) CreateUser(ctx context.Context, user *models.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}
```

### Test Service Factory

```go
// From internal/auth/service_test.go
func newTestService(t *testing.T, repo RepositoryInterface) *Service {
    t.Helper()
    manager, err := jwtkeys.NewManager(context.Background(), jwtkeys.Config{
        RotationInterval: 365 * 24 * time.Hour,
        GracePeriod:      365 * 24 * time.Hour,
        LegacySecret:     "test-secret",
    })
    if err != nil {
        t.Fatalf("failed to create jwt manager: %v", err)
    }
    return NewService(repo, manager, 24)
}
```

### Assertion Helpers

```go
// From test/helpers/assertions.go
func AssertPasswordNotInResponse(t *testing.T, user *models.User) {
    t.Helper()
    // Password hash should not be exposed in API responses
    assert.Empty(t, user.PasswordHash, "password hash should not be in response")
}

func AssertValidJWT(t *testing.T, tokenString string) {
    t.Helper()
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return []byte("test-secret"), nil
    })
    assert.NoError(t, err)
    assert.True(t, token.Valid)
}

func AssertUserEqual(t *testing.T, expected, actual *models.User) {
    t.Helper()
    assert.Equal(t, expected.ID, actual.ID)
    assert.Equal(t, expected.Email, actual.Email)
    assert.Equal(t, expected.FirstName, actual.FirstName)
    assert.Equal(t, expected.LastName, actual.LastName)
}
```

### Integration Tests

```go
// From test/integration/e2e_ride_flow_test.go
func TestE2ERideFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Full ride lifecycle: request -> match -> pickup -> complete -> rate
    // Uses real database connections via test containers
}
```

### Coverage Statistics

| Metric | Value |
|--------|-------|
| Test files | 90+ |
| Test functions | 2,637+ |
| Line coverage | ~90% |
| Critical path coverage | 100% |

## Consequences

### Positive

- **Comprehensive coverage**: 2,600+ test functions cover most code paths.
- **Fast execution**: Unit tests complete in seconds (no I/O).
- **Clear failures**: Table-driven tests pinpoint exact failure case.
- **Reusable fixtures**: Shared helpers reduce test boilerplate.
- **Mock verification**: `AssertExpectations` catches unused mock setups.

### Negative

- **Mock maintenance**: Interface changes require mock updates.
- **Test file size**: Comprehensive tests make files lengthy.
- **False confidence**: Mocks may not catch integration issues.

### Running Tests

```bash
# Run all unit tests
make test

# Run with coverage
make test-coverage

# Run integration tests
make test-integration

# Run specific package
go test ./internal/auth/... -v
```

## References

- [internal/auth/service_test.go](/internal/auth/service_test.go) - Service unit tests
- [internal/auth/handler_test.go](/internal/auth/handler_test.go) - Handler tests
- [test/mocks/repository.go](/test/mocks/repository.go) - Mock implementations
- [test/helpers/fixtures.go](/test/helpers/fixtures.go) - Test fixtures
- [test/helpers/assertions.go](/test/helpers/assertions.go) - Custom assertions
- [test/integration/](/test/integration/) - Integration tests
