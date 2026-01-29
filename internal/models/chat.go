package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Conversation represents a chat thread between users
// For V1, this is largely virtual/derived from messages, but good to have for efficient querying if we expand
type Conversation struct {
	ID        string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	IsGroup   bool   `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time

	// Relations
	Participants []User    `gorm:"many2many:conversation_participants;"`
	Messages     []Message `gorm:"foreignKey:ConversationID"`
}

type Message struct {
	ID             string  `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	ConversationID *string `gorm:"index;type:uuid"` // Optional for V1 if we just do direct messages by UserID
	SenderID       string  `gorm:"index;type:text;not null"`
	RecipientID    string  `gorm:"index;type:text;not null"` // Redundant if using ConversationID, but good for DMs
	Content        string  `gorm:"type:text;not null"`
	IsRead         bool    `gorm:"default:false"`
	ReadAt         *time.Time
	CreatedAt      time.Time

	// Relations
	Sender    User `gorm:"foreignKey:SenderID"`
	Recipient User `gorm:"foreignKey:RecipientID"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return
}

// ConversationParticipant tracks who is in a conversation
type ConversationParticipant struct {
	ConversationID string `gorm:"primaryKey;type:uuid"`
	UserID         string `gorm:"primaryKey;type:text"`
	JoinedAt       time.Time
}
