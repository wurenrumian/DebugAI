package models

import (
	"time"

	"gorm.io/gorm"
)

// Conversation represents a debug conversation session
type Conversation struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	ConversationID string         `json:"conversation_id" gorm:"uniqueIndex;not null"` //  conversation ID
	StudentID      string         `json:"student_id" gorm:"index;not null"`            // student ID
	TaskType       string         `json:"task_type" gorm:"default:debug"`              // task type: debug, evaluate, recommend
	IsClosed       bool           `json:"is_closed" gorm:"default:false"`              // whether the conversation is closed
	ClosedAt       *time.Time     `json:"closed_at,omitempty"`                         // when the conversation was closed
}

// TableName specifies the table name for Conversation model
func (Conversation) TableName() string {
	return "conversations"
}
