package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestListChatContacts(t *testing.T) {
	SetupTestDB() // Re-uses setup from arena_test if in same package
	gin.SetMode(gin.TestMode)

	// Users
	me := models.User{ID: "me_chat", Username: "me_chat", Email: "me_chat@example.com"}
	u1 := models.User{ID: "u1_chat", Username: "user1_chat", Email: "u1_chat@example.com"} // Old message
	u2 := models.User{ID: "u2_chat", Username: "user2_chat", Email: "u2_chat@example.com"} // Recent message
	u3 := models.User{ID: "u3_chat", Username: "user3_chat", Email: "u3_chat@example.com"} // No message
	database.DB.Create(&me)
	database.DB.Create(&u1)
	database.DB.Create(&u2)
	database.DB.Create(&u3)

	// Messages
	// u1 sent message long ago
	database.DB.Create(&models.Message{ID: "m1", SenderID: "u1_chat", ReceiverID: "me_chat", Content: "Old", CreatedAt: time.Now().Add(-2 * time.Hour)})
	// u2 sent message recently
	database.DB.Create(&models.Message{ID: "m2", SenderID: "me_chat", ReceiverID: "u2_chat", Content: "Recent", CreatedAt: time.Now().Add(-1 * time.Minute)})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/chat/contacts", nil)
	c.Set("userId", "me_chat")

	ListChatContacts(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Contacts []models.User `json:"contacts"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)

	// Should contain u1 and u2, but NOT u3
	assert.Len(t, response.Contacts, 2)

	// Order should be u2 (Recent) then u1 (Old)
	// Check range to avoid panic if empty
	if len(response.Contacts) >= 2 {
		assert.Equal(t, "u2_chat", response.Contacts[0].ID)
		assert.Equal(t, "u1_chat", response.Contacts[1].ID)
	}
}
