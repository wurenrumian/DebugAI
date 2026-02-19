package models

import (
	"gorm.io/gorm"
)

// AIRecord represents an AI interaction record in the database
type AIRecord struct {
	gorm.Model
	ConversationID  string `json:"conversation_id" gorm:"index"`      //  the conversation ID
	StudentID       string `json:"student_id" gorm:"index"`           //  the student ID
	RoundNumber     int    `json:"round_number"`                      //  round number (1-4)
	Role            string `json:"role"`                              //  role: "student" or "assistant"
	RequestPayload  string `json:"request_payload" gorm:"type:text"`  //  the original request JSON sent to AI service
	ResponsePayload string `json:"response_payload" gorm:"type:text"` //  the original response JSON received from AI service
	Error           string `json:"error,omitempty" gorm:"type:text"`  //  error message if any
}
