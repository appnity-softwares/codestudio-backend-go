package models

import "time"

// Message represents a direct message between users
type Message struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	SenderID      string    `json:"senderId" gorm:"index"`
	Sender        User      `json:"sender" gorm:"foreignKey:SenderID"`
	ReceiverID    string    `json:"receiverId" gorm:"index"`
	Receiver      User      `json:"receiver" gorm:"foreignKey:ReceiverID"`
	Content       string    `json:"content"`
	AttachmentURL string    `json:"attachmentUrl"`
	IsRead        bool      `json:"isRead" gorm:"default:false"`
	CreatedAt     time.Time `json:"createdAt"`
}
