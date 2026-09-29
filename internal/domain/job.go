package domain

import (
	"time"
)

// JobRecord represents a persistent background job in the database queue
type JobRecord struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Queue       string     `gorm:"size:100;index:idx_queue_status_avail;not null;default:'default'" json:"queue"`
	Name        string     `gorm:"size:255;not null" json:"name"`
	Payload     string     `gorm:"type:text;not null" json:"payload"`
	Attempts    int        `gorm:"not null;default:0" json:"attempts"`
	MaxAttempts int        `gorm:"not null;default:3" json:"max_attempts"`
	ReservedAt  *time.Time `json:"reserved_at"`
	AvailableAt time.Time  `gorm:"index:idx_queue_status_avail;not null" json:"available_at"`
	Status      string     `gorm:"size:30;index:idx_queue_status_avail;not null;default:'pending'" json:"status"` // pending, processing, failed, completed
	LastError   string     `gorm:"type:text" json:"last_error"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TableName overrides the default table name to "jobs" (Laravel convention)
func (JobRecord) TableName() string {
	return "jobs"
}

// Queue Job Identifiers
const (
	JobWelcomeEmail = "email:welcome"
)

// WelcomeEmailPayload holds parameters for sending welcome emails asynchronously
type WelcomeEmailPayload struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	UserName string `json:"user_name"`
}
