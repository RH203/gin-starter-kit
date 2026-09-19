package domain

import (
	"context"
	"time"
)

// User represents the domain model entity
type User struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name      string    `json:"name" gorm:"type:varchar(100);not null"`
	Email     string    `json:"email" gorm:"type:varchar(150);uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DTOs for HTTP Delivery & Usecase
type CreateUserRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type UpdateUserRequest struct {
	Name  string `json:"name" binding:"omitempty,min=2,max=100"`
	Email string `json:"email" binding:"omitempty,email"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string        `json:"-"`
	User  *UserResponse `json:"user"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// UserRepository interface defines data access contracts
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindAll(ctx context.Context, limit, offset int) ([]User, int64, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}

// UserUsecase interface defines business logic contracts
type UserUsecase interface {
	Register(ctx context.Context, req *CreateUserRequest) (*UserResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
	GetProfile(ctx context.Context, userID string) (*UserResponse, error)
	GetByID(ctx context.Context, id string) (*UserResponse, error)
	GetAll(ctx context.Context, page, pageSize int) ([]UserResponse, int64, error)
	Update(ctx context.Context, id string, req *UpdateUserRequest) (*UserResponse, error)
	Delete(ctx context.Context, id string) error
}
