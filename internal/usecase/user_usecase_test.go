package usecase_test

import (
	"context"
	"testing"

	"gin-starter-pack/config"
	"gin-starter-pack/internal/domain"
	"gin-starter-pack/internal/usecase"
	"gin-starter-pack/pkg/jwt"
	"gin-starter-pack/pkg/redis"
)

// MockUserRepository implements domain.UserRepository in-memory
type mockUserRepository struct {
	users map[string]*domain.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{users: make(map[string]*domain.User)}
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	u, exists := m.users[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.User, int64, error) {
	var list []domain.User
	for _, u := range m.users {
		list = append(list, *u)
	}
	return list, int64(len(list)), nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id string) error {
	if _, exists := m.users[id]; !exists {
		return domain.ErrNotFound
	}
	delete(m.users, id)
	return nil
}

func TestUserUsecase_Register_Success(t *testing.T) {
	repo := newMockUserRepository()
	cache, _ := redis.InitRedis(&config.RedisConfig{Enabled: false})
	jwtService := jwt.NewJWTService("test-secret", 24)

	uc := usecase.NewUserUsecase(repo, cache, jwtService, nil)

	req := &domain.CreateUserRequest{
		Name:     "Budi Gunawan",
		Email:    "budi@example.com",
		Password: "secretpassword",
	}

	resp, err := uc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID == "" {
		t.Errorf("expected generated ID, got empty string")
	}
	if resp.Email != req.Email {
		t.Errorf("expected email %s, got %s", req.Email, resp.Email)
	}
}

func TestUserUsecase_Register_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepository()
	cache, _ := redis.InitRedis(&config.RedisConfig{Enabled: false})
	jwtService := jwt.NewJWTService("test-secret", 24)

	uc := usecase.NewUserUsecase(repo, cache, jwtService, nil)

	req := &domain.CreateUserRequest{
		Name:     "Budi Gunawan",
		Email:    "budi@example.com",
		Password: "secretpassword",
	}

	_, err := uc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("expected first register to succeed, got %v", err)
	}

	// Try register again with same email
	_, err = uc.Register(context.Background(), req)
	if err != domain.ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestUserUsecase_Login_Success(t *testing.T) {
	repo := newMockUserRepository()
	cache, _ := redis.InitRedis(&config.RedisConfig{Enabled: false})
	jwtService := jwt.NewJWTService("test-secret", 24)

	uc := usecase.NewUserUsecase(repo, cache, jwtService, nil)

	// Register user first
	_, err := uc.Register(context.Background(), &domain.CreateUserRequest{
		Name:     "Siti Aminah",
		Email:    "siti@example.com",
		Password: "mysecretpassword",
	})
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	// Login
	loginResp, err := uc.Login(context.Background(), &domain.LoginRequest{
		Email:    "siti@example.com",
		Password: "mysecretpassword",
	})
	if err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}

	if loginResp.Token == "" {
		t.Errorf("expected non-empty JWT token")
	}
	if loginResp.User.Email != "siti@example.com" {
		t.Errorf("expected email siti@example.com, got %s", loginResp.User.Email)
	}
}

func TestUserUsecase_Login_InvalidPassword(t *testing.T) {
	repo := newMockUserRepository()
	cache, _ := redis.InitRedis(&config.RedisConfig{Enabled: false})
	jwtService := jwt.NewJWTService("test-secret", 24)

	uc := usecase.NewUserUsecase(repo, cache, jwtService, nil)

	_, _ = uc.Register(context.Background(), &domain.CreateUserRequest{
		Name:     "Siti Aminah",
		Email:    "siti2@example.com",
		Password: "mysecretpassword",
	})

	_, err := uc.Login(context.Background(), &domain.LoginRequest{
		Email:    "siti2@example.com",
		Password: "wrongpassword",
	})
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestUserUsecase_GetByID_NotFound(t *testing.T) {
	repo := newMockUserRepository()
	cache, _ := redis.InitRedis(&config.RedisConfig{Enabled: false})
	jwtService := jwt.NewJWTService("test-secret", 24)

	uc := usecase.NewUserUsecase(repo, cache, jwtService, nil)

	_, err := uc.GetByID(context.Background(), "non-existent-id")
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
