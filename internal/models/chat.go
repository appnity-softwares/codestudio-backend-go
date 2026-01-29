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
	// Primary Key - UUID stored as string (consistent with rest of codebase)
	// Using type:text to match legacy schema
	ID string `gorm:"primaryKey;type:text;default:uuid_generate_v4()" json:"id"`

	// Conversation (optional for DMs)
	ConversationID *string `gorm:"index;type:text" json:"conversationId"`

	// Core Fields - SenderID/RecipientID are User IDs (text type from User model)
	SenderID    string `gorm:"index;type:text;not null" json:"senderId"`
	RecipientID string `gorm:"index;type:text;not null" json:"recipientId"`
	Content     string `gorm:"type:text;not null" json:"content"`

	// Message Type: text, code, image, system
	Type string `gorm:"type:text;default:'text';not null" json:"type"`

	// Delivery Status: sending, sent, delivered, read, failed
	Status string `gorm:"type:text;default:'sent';not null" json:"status"`

	// Read Tracking
	IsRead bool       `gorm:"default:false" json:"isRead"`
	ReadAt *time.Time `json:"readAt"`

	// Timestamps
	CreatedAt time.Time      `json:"createdAt"`
	EditedAt  *time.Time     `json:"editedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // Use GORM soft delete

	// Idempotency Key (client-generated UUID for deduplication)
	// Using type:uuid to match ID column type for internal consistency
	ClientMessageID *string `gorm:"index;type:uuid" json:"clientMessageId"`

	// Threading/Replies
	// ReplyToID must match the type of the ID column in the DB.
	// We suspect ID is 'text' in the DB (legacy), so we set this to 'text' to avoid
	// GORM trying to alter it to 'uuid' which breaks the FK constraint.
	ReplyToID *string  `gorm:"type:text;index" json:"replyToId"`
	ReplyTo   *Message `gorm:"-" json:"replyTo,omitempty"`

	// Metadata (JSON: reactions, mentions, code language, etc.)
	Metadata string `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	// Relations
	Sender    User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Recipient User `gorm:"foreignKey:RecipientID" json:"recipient,omitempty"`
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

// ============================================
// PHASE 7: REACTIONS, MENTIONS
// ============================================

// MessageReaction stores emoji reactions on messages
type MessageReaction struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	MessageID string    `gorm:"index;type:text;not null" json:"messageId"` // changed to text to match legacy Message ID type
	UserID    string    `gorm:"index;type:text;not null" json:"userId"`
	Emoji     string    `gorm:"type:text;not null" json:"emoji"` // e.g., "👍", "❤️", "😂"
	CreatedAt time.Time `json:"createdAt"`

	// Relations
	Message Message `gorm:"foreignKey:MessageID" json:"-"`
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (r *MessageReaction) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return
}

// UniqueIndex: one reaction per emoji per user per message
// CREATE UNIQUE INDEX idx_unique_reaction ON message_reactions(message_id, user_id, emoji);

// Mention tracks @mentions in messages
type Mention struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	MessageID   string    `gorm:"index;type:text;not null" json:"messageId"`   // changed to text
	MentionedID string    `gorm:"index;type:text;not null" json:"mentionedId"` // User who was mentioned
	MentionerID string    `gorm:"index;type:text;not null" json:"mentionerId"` // User who mentioned
	StartIndex  int       `json:"startIndex"`                                  // Position in message content
	EndIndex    int       `json:"endIndex"`
	CreatedAt   time.Time `json:"createdAt"`

	// Relations
	Message   Message `gorm:"foreignKey:MessageID" json:"-"`
	Mentioned User    `gorm:"foreignKey:MentionedID" json:"mentioned,omitempty"`
	Mentioner User    `gorm:"foreignKey:MentionerID" json:"-"`
}

func (m *Mention) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return
}

// Allowed emojis for reactions (curated list for consistency)
var AllowedReactionEmojis = []string{
	"👍", "👎", "❤️", "😂", "😮", "😢", "🔥", "🎉", "🚀", "👀",
}

// IsValidReactionEmoji checks if an emoji is in the allowed list
func IsValidReactionEmoji(emoji string) bool {
	for _, e := range AllowedReactionEmojis {
		if e == emoji {
			return true
		}
	}
	return false
}
