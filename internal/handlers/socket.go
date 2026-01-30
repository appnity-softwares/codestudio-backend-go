package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/pkg/utils"
)

var SocketServer *socketio.Server

// Presence tracking
var (
	onlineUsers   = make(map[string]string) // userId -> socketId
	onlineUsersMu sync.RWMutex
)

// Typing throttle: track last typing emit per user to prevent spam
var (
	lastTypingEmit         = make(map[string]time.Time) // userId -> last emit time
	lastTypingMu           sync.RWMutex
	typingThrottleDuration = 3 * time.Second // Minimum interval between typing events
)

// GetOnlineUsers returns list of online user IDs
func GetOnlineUsers() []string {
	onlineUsersMu.RLock()
	defer onlineUsersMu.RUnlock()

	users := make([]string, 0, len(onlineUsers))
	for userId := range onlineUsers {
		users = append(users, userId)
	}
	return users
}

// IsUserOnline checks if a user is online
func IsUserOnline(userId string) bool {
	onlineUsersMu.RLock()
	defer onlineUsersMu.RUnlock()
	_, exists := onlineUsers[userId]
	return exists
}

// SendNotificationToUser sends a real-time notification to a specific user
func SendNotificationToUser(userId string, notification map[string]interface{}) {
	if SocketServer != nil {
		SocketServer.BroadcastToRoom("/", userId, "notification", notification)
	}
}

// BroadcastPresenceUpdate broadcasts user online/offline status to all clients
func BroadcastPresenceUpdate(userId string, isOnline bool) {
	if SocketServer != nil {
		data := map[string]interface{}{
			"userId":   userId,
			"isOnline": isOnline,
		}
		SocketServer.BroadcastToRoom("/", "presence", "presence_update", data)
	}
}

func InitSocketServer() *socketio.Server {
	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&websocket.Transport{
				CheckOrigin: func(r *http.Request) bool { return true },
			},
			&polling.Transport{
				CheckOrigin: func(r *http.Request) bool { return true },
			},
		},
	})

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		url := s.URL()

		// 1. Try to get token from Query Param (most reliable for ws handshake)
		token := url.Query().Get("token")

		// 2. Validate Token
		if token == "" {
			token = url.Query().Get("auth_token") // Fallback
		}

		if token == "" {
			log.Println("Socket Connection Rejected: No token provided", s.ID())
			return fmt.Errorf("authentication required")
		}

		claims, err := utils.ValidateToken(token)
		if err != nil {
			log.Println("Socket Connection Rejected: Invalid token", s.ID())
			return fmt.Errorf("invalid token")
		}

		userId := claims.UserID
		log.Println("Socket authenticated:", s.ID(), "User:", userId)

		// Optimization: Store userId directly in socket context for O(1) lookup
		s.SetContext(userId)

		// Track user as online
		onlineUsersMu.Lock()
		onlineUsers[userId] = s.ID()
		onlineUsersMu.Unlock()

		// Join personal room for notifications
		s.Join(userId)

		// Join global presence room
		s.Join("presence")

		// Broadcast that user is online
		BroadcastPresenceUpdate(userId, true)

		// Send current online users list to the connecting user
		s.Emit("online_users", GetOnlineUsers())

		return nil
	})

	server.OnEvent("/", "join_chat", func(s socketio.Conn, chatId string) {
		log.Println("User joined chat:", chatId)
		s.Join(chatId)
	})

	server.OnEvent("/", "typing", func(s socketio.Conn, data map[string]interface{}) {
		recipientID, ok := data["recipientId"].(string)
		if !ok {
			recipientID, _ = data["receiverId"].(string)
		}

		if recipientID != "" {
			// Find who is typing (O(1) from socket context)
			senderID, _ := s.Context().(string)
			if senderID == "" {
				return
			}

			// THROTTLE: Only emit if 3s since last emit for this sender
			lastTypingMu.RLock()
			lastTime, exists := lastTypingEmit[senderID]
			lastTypingMu.RUnlock()

			if exists && time.Since(lastTime) < typingThrottleDuration {
				return // Throttled - skip this event
			}

			lastTypingMu.Lock()
			lastTypingEmit[senderID] = time.Now()
			lastTypingMu.Unlock()

			server.BroadcastToRoom("/", recipientID, "user_typing", map[string]interface{}{
				"userId":    senderID,
				"expiresAt": time.Now().Add(4 * time.Second).Unix(), // Auto-expire on client
			})
		}
	})

	// Get online users request
	server.OnEvent("/", "get_online_users", func(s socketio.Conn, msg string) {
		s.Emit("online_users", GetOnlineUsers())
	})

	// Message ACK event - client sends after receiving/reading message
	server.OnEvent("/", "message_ack", func(s socketio.Conn, data map[string]interface{}) {
		messageID, _ := data["messageId"].(string)
		status, _ := data["status"].(string) // "delivered" or "read"

		if messageID == "" || status == "" {
			return
		}

		// Validate status
		if status != "delivered" && status != "read" {
			return
		}

		// Notify sender that message was delivered/read
		// Use minimal DB hit: only get sender_id
		var senderID string
		if err := database.DB.Table("messages").Select("sender_id").Where("id = ?", messageID).Scan(&senderID).Error; err == nil && senderID != "" {
			// Update status in background to respond faster to socket
			go func(mid string, st string) {
				updates := map[string]interface{}{"status": st}
				if st == "read" {
					now := time.Now()
					updates["is_read"] = true
					updates["read_at"] = &now
				}
				database.DB.Table("messages").Where("id = ?", mid).Updates(updates)
			}(messageID, status)

			server.BroadcastToRoom("/", senderID, "message_status", map[string]interface{}{
				"messageId": messageID,
				"status":    status,
			})
		}
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("closed", reason)

		// Find and remove user from online list
		onlineUsersMu.Lock()
		var disconnectedUserId string
		for userId, socketId := range onlineUsers {
			if socketId == s.ID() {
				disconnectedUserId = userId
				delete(onlineUsers, userId)
				break
			}
		}
		onlineUsersMu.Unlock()

		// Broadcast that user is offline
		if disconnectedUserId != "" {
			BroadcastPresenceUpdate(disconnectedUserId, false)
		}
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("meet error:", e)
	})

	go server.Serve()
	SocketServer = server
	return server
}

// Gin Handler to wrap Socket.io
func SocketHandler(server *socketio.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		server.ServeHTTP(c.Writer, c.Request)
	}
}
