package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/pkg/utils"
	razorpay "github.com/razorpay/razorpay-go"
)

type CreateOrderInput struct {
	EventID string `json:"eventId" binding:"required"`
}

type VerifyPaymentInput struct {
	EventID           string `json:"eventId" binding:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
	RazorpayOrderID   string `json:"razorpay_order_id" binding:"required"`
	RazorpaySignature string `json:"razorpay_signature" binding:"required"`
}

func CreateOrder(c *gin.Context) {
	var input CreateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch Event
	var event models.Event
	if err := database.DB.Where("id = ?", input.EventID).First(&event).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	if event.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event is free, no payment needed"})
		return
	}

	keyID := os.Getenv("RAZORPAY_KEY_ID")
	keySecret := os.Getenv("RAZORPAY_KEY_SECRET")

	if keyID == "" || keySecret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment gateway not configured"})
		return
	}

	client := razorpay.NewClient(keyID, keySecret)

	amountInPaise := event.Price * 100
	data := map[string]interface{}{
		"amount":   amountInPaise,
		"currency": "INR",
		"receipt":  "receipt_" + input.EventID,
	}

	body, err := client.Order.Create(data, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order", "details": err.Error()})
		return
	}

	orderID, _ := body["id"].(string)

	c.JSON(http.StatusOK, gin.H{
		"orderId":  orderID,
		"amount":   amountInPaise,
		"currency": "INR",
		"keyId":    keyID,
	})
}

func VerifyPayment(c *gin.Context) {
	var input VerifyPaymentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	keySecret := os.Getenv("RAZORPAY_KEY_SECRET")

	// signature verification
	data := input.RazorpayOrderID + "|" + input.RazorpayPaymentID
	h := hmac.New(sha256.New, []byte(keySecret))
	h.Write([]byte(data))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	if expectedSignature != input.RazorpaySignature {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature"})
		return
	}

	// Update Registration Status
	userID := c.MustGet("userId").(string)

	var registration models.Registration
	err := database.DB.Where("user_id = ? AND event_id = ?", userID, input.EventID).First(&registration).Error

	if err != nil {
		registration = models.Registration{
			ID:        utils.GenerateID(),
			UserID:    userID,
			EventID:   input.EventID,
			PaymentID: input.RazorpayPaymentID,
			Status:    models.RegStatusPaid,
			CreatedAt: time.Now(),
		}
		if result := database.DB.Create(&registration); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save registration"})
			return
		}
	} else {
		registration.Status = models.RegStatusPaid
		registration.PaymentID = input.RazorpayPaymentID
		database.DB.Save(&registration)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Payment verified and registered"})
}

// Handler for Webhook if needed (e.g. async confirmation)
// Currently VerifyPayment is synchronous via frontend callback.
func HandleRazorpayWebhook(c *gin.Context) {
	// Basic stub or future implementation.
	// Usually webhooks verify signature header 'X-Razorpay-Signature'.
	c.Status(http.StatusOK)
}
