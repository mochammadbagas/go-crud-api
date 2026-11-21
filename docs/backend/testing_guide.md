# Automated Testing Standards & Strategy

**Document Type:** Testing Standards & Quality Assurance Architecture  
**Target Audience:** Backend Developers & QA Engineers  
**Target Metric:** Zero-Flake, Fast (<2s total suite), Isolated Unit & Integration Tests

---

## 1. Testing Philosophy: Zero Database Coupling

To guarantee deterministic, instantaneous test execution:

1. **Domain Services are Tested in Pure Isolation**: Business logic (`internal/service/*_test.go`) never connects to a live MySQL instance or network port.
2. **Mock Repositories via Interfaces**: Services declare what they need through Go interfaces. Unit tests pass lightweight in-memory implementations.
3. **HTTP Handlers Tested via `httptest`**: HTTP controllers (`internal/handlers/*_test.go`) use standard library `net/http/httptest` recorders.

---

## 2. In-Memory Mock Repository Pattern

Because Go interfaces are satisfied implicitly, creating test mocks requires no bulky third-party mock generation libraries:

```go
// Example Mock User Repository for testing AuthService
type mockUserRepo struct {
    users map[string]*models.User
}

func newMockUserRepo() *mockUserRepo {
    return &mockUserRepo{users: make(map[string]*models.User)}
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
    for _, u := range m.users {
        if u.Email == email {
            return u, nil
        }
    }
    return nil, repository.ErrNotFound
}

func (m *mockUserRepo) Create(ctx context.Context, user *models.User) error {
    m.users[user.ID] = user
    return nil
}
```

---

## 3. Table-Driven Test Structure

All tests follow idiomatic Go **Table-Driven Testing**:

```go
func TestAuthService_Login(t *testing.T) {
    tests := []struct {
        name          string
        email         string
        password      string
        setupMock     func(repo *mockUserRepo)
        expectedError error
    }{
        {
            name:     "successful login",
            email:    "user@example.com",
            password: "validpassword123",
            setupMock: func(repo *mockUserRepo) {
                hash, _ := utils.HashPassword("validpassword123")
                repo.users["01ARZ..."] = &models.User{
                    ID:           "01ARZ...",
                    Email:        "user@example.com",
                    PasswordHash: hash,
                    Status:       models.UserStatusActive,
                }
            },
            expectedError: nil,
        },
        {
            name:          "user not found",
            email:         "unknown@example.com",
            password:      "password",
            setupMock:     func(repo *mockUserRepo) {},
            expectedError: service.ErrInvalidCredentials,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := newMockUserRepo()
            tt.setupMock(repo)

            svc := service.NewAuthService(repo, ...)
            _, err := svc.Login(context.Background(), tt.email, tt.password)

            if !errors.Is(err, tt.expectedError) {
                t.Fatalf("expected error %v, got %v", tt.expectedError, err)
            }
        })
    }
}
```

---

## 4. Test Execution Commands

### Run Full Test Suite

```bash
go test -v -count=1 ./...
```

### Run with Race Detector

Detects concurrent read/write data races across goroutines:

```bash
go test -race ./...
```

### Run Specific Package Tests

```bash
go test -v ./internal/service/...
```

### Coverage Reporting

```bash
go test -cover -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```
