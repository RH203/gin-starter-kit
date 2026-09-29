package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"gin-starter-pack/internal/domain"
	"gin-starter-pack/pkg/hash"
	"gin-starter-pack/pkg/jwt"
	"gin-starter-pack/pkg/mail"
	"gin-starter-pack/pkg/queue"
	"gin-starter-pack/pkg/redis"
	"gin-starter-pack/pkg/worker"

	"github.com/google/uuid"
)

type userUsecase struct {
	userRepo domain.UserRepository
	cache    *redis.Client
	jwt      *jwt.Service
	worker   *worker.Pool
	queue    queue.Queue
	mailer   mail.Mailer
}

// NewUserUsecase creates a new UserUsecase
func NewUserUsecase(
	userRepo domain.UserRepository,
	cache *redis.Client,
	jwtService *jwt.Service,
	queueDispatcher queue.Queue,
	mailer ...mail.Mailer,
) domain.UserUsecase {
	var m mail.Mailer
	if len(mailer) > 0 {
		m = mailer[0]
	}
	return &userUsecase{
		userRepo: userRepo,
		cache:    cache,
		jwt:      jwtService,
		queue:    queueDispatcher,
		mailer:   m,
	}
}

// RegisterUserQueueHandlers registers domain worker handlers for user events
func RegisterUserQueueHandlers(reg *queue.Registry, mailer mail.Mailer) {
	reg.Register(domain.JobWelcomeEmail, func(ctx context.Context, payload []byte) error {
		var data domain.WelcomeEmailPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return err
		}

		if mailer != nil {
			subject := fmt.Sprintf("Welcome to Gin Starter Pack, %s!", data.UserName)
			templateData := map[string]interface{}{
				"AppName":   "Gin Starter Pack",
				"UserName":  data.UserName,
				"Email":     data.Email,
				"ActionURL": "http://localhost:8080",
				"Year":      time.Now().Year(),
			}
			return mailer.SendTemplate(ctx, data.Email, subject, "welcome.html", templateData)
		}

		slog.Info("Welcome email dispatched to user", "email", data.Email, "name", data.UserName, "user_id", data.UserID)
		return nil
	})
}

func (u *userUsecase) Register(ctx context.Context, req *domain.CreateUserRequest) (*domain.UserResponse, error) {
	// Check if email already exists
	existingUser, err := u.userRepo.FindByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, domain.ErrAlreadyExists
	}

	// Hash password using dedicated hash package
	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Email:     req.Email,
		Password:  hashedPassword,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Dispatch asynchronous background job (e.g. welcome email)
	if u.queue != nil {
		_ = u.queue.Dispatch(ctx, domain.JobWelcomeEmail, domain.WelcomeEmailPayload{
			UserID:   user.ID,
			Email:    user.Email,
			UserName: user.Name,
		})
	} else if u.worker != nil {
		u.worker.Dispatch(&worker.WelcomeEmailJob{
			UserID:   user.ID,
			Email:    user.Email,
			UserName: user.Name,
			Mailer:   u.mailer,
		})
	}

	return user.ToResponse(), nil
}

func (u *userUsecase) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	user, err := u.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Verify password using dedicated hash package
	if !hash.CheckPasswordHash(req.Password, user.Password) {
		return nil, domain.ErrInvalidCredentials
	}

	// Generate JWT token
	token, err := u.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	return &domain.LoginResponse{
		Token: token,
		User:  user.ToResponse(),
	}, nil
}

func (u *userUsecase) GetProfile(ctx context.Context, userID string) (*domain.UserResponse, error) {
	return u.GetByID(ctx, userID)
}

func (u *userUsecase) GetByID(ctx context.Context, id string) (*domain.UserResponse, error) {
	cacheKey := fmt.Sprintf("user:%s", id)

	return redis.Remember(ctx, u.cache, cacheKey, 10*time.Minute, func() (*domain.UserResponse, error) {
		user, err := u.userRepo.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return user.ToResponse(), nil
	})
}

func (u *userUsecase) GetAll(ctx context.Context, page, pageSize int) ([]domain.UserResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	users, total, err := u.userRepo.FindAll(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]domain.UserResponse, len(users))
	for i, user := range users {
		responses[i] = *user.ToResponse()
	}

	return responses, total, nil
}

func (u *userUsecase) Update(ctx context.Context, id string, req *domain.UpdateUserRequest) (*domain.UserResponse, error) {
	user, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" && req.Email != user.Email {
		// Check email unique
		existing, err := u.userRepo.FindByEmail(ctx, req.Email)
		if err == nil && existing != nil && existing.ID != id {
			return nil, domain.ErrAlreadyExists
		}
		user.Email = req.Email
	}
	user.UpdatedAt = time.Now().UTC()

	if err := u.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Invalidate user cache
	_ = u.cache.Invalidate(ctx, fmt.Sprintf("user:%s", id))

	return user.ToResponse(), nil
}

func (u *userUsecase) Delete(ctx context.Context, id string) error {
	if err := u.userRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate user cache
	_ = u.cache.Invalidate(ctx, fmt.Sprintf("user:%s", id))

	return nil
}
