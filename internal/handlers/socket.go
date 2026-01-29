package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
	"github.com/pushp314/devconnect-backend/pkg/utils"
)

var SocketServer *socketio.Server

// Presence tracking
var (
	onlineUsers   = make(map[string]string) // userId -> socketId
	onlineUsersMu sync.RWMutex
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
			&polling.Transport{
				CheckOrigin: func(r *http.Request) bool { return true },
			},
			&websocket.Transport{
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
			log.Println("Socket Connection Rejected: Invalid token", s.ID(), err)
			return fmt.Errorf("invalid token")
		}

		userId := claims.UserID
		log.Println("Socket authenticated:", s.ID(), "User:", userId)

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
			// Find who is typing (from socket context or map)
			senderID := ""
			onlineUsersMu.RLock()
			for uid, sid := range onlineUsers {
				if sid == s.ID() {
					senderID = uid
					break
				}
			}
			onlineUsersMu.RUnlock()

			if senderID != "" {
				server.BroadcastToRoom("/", recipientID, "user_typing", map[string]interface{}{
					"userId": senderID,
				})
			}
		}
	})

	// Get online users request
	server.OnEvent("/", "get_online_users", func(s socketio.Conn, msg string) {
		s.Emit("online_users", GetOnlineUsers())
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
