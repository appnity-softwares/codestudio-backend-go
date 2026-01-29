package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/config"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/pkg/logger"
	"github.com/pushp314/devconnect-backend/pkg/utils"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

// --- Helper Functions ---

func validatePasswordStrength(password string) error {
	var (
		hasMinLen  = false
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	if len(password) >= 8 {
		hasMinLen = true
	}
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	if !hasMinLen || !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return fmt.Errorf("password must be at least 8 characters long and contain at least one uppercase letter, one lowercase letter, one number, and one special character")
	}
	return nil
}

// --- Local Auth ---

type RegisterInput struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Username string `json:"username" binding:"required"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate Password Strength
	if err := validatePasswordStrength(input.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate Username
	if !utils.ValidateUsername(input.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username must be 3-30 characters and contain only letters, numbers, underscores, or hyphens (no spaces or @ allowed)"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Name:     input.Name,
		Email:    input.Email,
		Username: input.Username,
		Password: string(hashedPassword),
	}

	if result := database.DB.Create(&user); result.Error != nil {
		// Differentiate between email and username conflict
		var existingUser models.User
		if err := database.DB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "An account with this email already exists. Please sign in instead."})
			return
		}
		if err := database.DB.Where("username = ?", input.Username).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "This username is already taken. Please choose another one."})
			return
		}

		logger.Warn().Err(result.Error).Str("email", input.Email).Msg("Registration failed: unique violation")
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email or username already exists"})
		return
	}

	// Generate Token
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	logger.Info().Str("user_id", user.ID).Msg("User registered successfully")

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user":  user,
	})
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if result := database.DB.Where("email = ?", input.Email).First(&user); result.Error != nil {
		logger.Warn().Str("email", input.Email).Msg("Login failed: user not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		logger.Warn().Str("email", input.Email).Msg("Login failed: invalid password")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	logger.Info().Str("user_id", user.ID).Msg("User logged in")

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

// Logout invalidates the token on the client side
func Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func CheckUsername(c *gin.Context) {
	username := c.Query("username")
	if len(username) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username too short"})
		return
	}

	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", username).Count(&count)

	if count > 0 {
		// Simple suggestion logic
		suggestions := []string{
			fmt.Sprintf("%s_dev", username),
			fmt.Sprintf("%s_code", username),
			fmt.Sprintf("%s%d", username, time.Now().Unix()%100),
		}
		c.JSON(http.StatusOK, gin.H{
			"available":   false,
			"suggestions": suggestions,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"available": true})
}

// --- OAuth ---

var (
	googleOauthConfig *oauth2.Config
	githubOauthConfig *oauth2.Config
)

func InitOAuthConfig() {
	if config.AppConfig.GoogleClientID != "" {
		googleOauthConfig = &oauth2.Config{
			RedirectURL:  config.AppConfig.GoogleCallbackURL,
			ClientID:     config.AppConfig.GoogleClientID,
			ClientSecret: config.AppConfig.GoogleClientSecret,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		}
	} else {
		logger.Warn().Msg("Google OAuth keys missing")
	}

	if config.AppConfig.GithubClientID != "" {
		githubOauthConfig = &oauth2.Config{
			RedirectURL:  config.AppConfig.GithubCallbackURL,
			ClientID:     config.AppConfig.GithubClientID,
			ClientSecret: config.AppConfig.GithubClientSecret,
			Scopes:       []string{"user:email", "read:user"},
			Endpoint:     github.Endpoint,
		}
	} else {
		logger.Warn().Msg("GitHub OAuth keys missing")
	}
}

// Google
func GoogleLogin(c *gin.Context) {
	if googleOauthConfig == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google OAuth not configured"})
		return
	}
	url := googleOauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func GoogleCallback(c *gin.Context) {
	if googleOauthConfig == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google OAuth not configured"})
		return
	}

	code := c.Query("code")
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		logger.Error().Err(err).Msg("Google OAuth exchange failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to exchange token: " + err.Error()})
		return
	}

	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get Google user info")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		logger.Error().Err(err).Msg("Failed to parse Google user info")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user info"})
		return
	}

	logger.Info().Str("email", userInfo.Email).Msg("Google user info retrieved successfully")
	user := handleOAuthLogin(c, userInfo.Email, userInfo.Name, userInfo.Picture)
	if user != nil {
		finishOAuthLogin(c, user)
	}
}

// GitHub
func GithubLogin(c *gin.Context) {
	if githubOauthConfig == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub OAuth not configured"})
		return
	}
	url := githubOauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func GithubCallback(c *gin.Context) {
	if githubOauthConfig == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub OAuth not configured"})
		return
	}

	code := c.Query("code")
	token, err := githubOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to exchange token"})
		return
	}

	client := githubOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID        int    `json:"id"`
		Login     string `json:"login"` // Username
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"` // Might be empty if private
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user info"})
		return
	}

	// If email is missing, fetch it
	email := userInfo.Email
	if email == "" {
		email = fmt.Sprintf("%s@github.placeholder", userInfo.Login) // Fallback for now
	}

	user := handleOAuthLogin(c, email, userInfo.Name, userInfo.AvatarURL)
	if user != nil && database.IsFeatureEnabled(models.SettingFeatureGithubStats) {
		// Fetch stats in background
		go func(tokenStr string, u models.User) {
			if err := FetchAndStoreGithubStats(tokenStr, &u); err != nil {
				logger.Error().Err(err).Str("user_id", u.ID).Msg("Failed to background sync GitHub stats")
			}
		}(token.AccessToken, *user)
	}

	if user != nil {
		finishOAuthLogin(c, user)
	}
}

// Common OAuth Handler - Resolves user by email or creates new
func handleOAuthLogin(c *gin.Context, email, name, image string) *models.User {
	var user models.User
	result := database.DB.Where("email = ?", email).First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		// New User logic
		logger.Info().Str("email", email).Msg("New user registration attempt via OAuth")

		var regSetting models.SystemSettings
		if err := database.DB.Where("key = ?", models.SettingRegistrationOpen).First(&regSetting).Error; err == nil {
			if regSetting.Value == "false" {
				logger.Warn().Str("email", email).Msg("Registration closed during OAuth attempt")
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "User registration is currently closed"})
				return nil
			}
		}

		// Generate better username from name or email prefix
		baseUsername := ""
		if name != "" {
			baseUsername = strings.ToLower(strings.ReplaceAll(name, " ", "_"))
		} else {
			baseUsername = strings.Split(email, "@")[0]
		}

		// Clean username
		cleaned := ""
		for _, r := range baseUsername {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
				cleaned += string(r)
			}
		}
		if cleaned == "" {
			cleaned = "user"
		}

		now := time.Now()
		user = models.User{
			ID:            uuid.New().String(),
			Email:         email,
			EmailVerified: &now,
			Name:          name,
			Image:         image,
			Username:      cleaned + "_" + uuid.New().String()[:4], // Ensure uniqueness
			Role:          models.RoleUser,
			Visibility:    models.VisibilityPublic,
		}

		if createErr := database.DB.Create(&user).Error; createErr != nil {
			logger.Error().Err(createErr).Str("email", email).Msg("CRITICAL: Failed to create user during OAuth")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Account creation failed",
				"details": createErr.Error(),
			})
			return nil
		}
		logger.Info().Str("email", email).Str("user_id", user.ID).Msg("New user successfully registered via OAuth")
	} else if result.Error != nil {
		logger.Error().Err(result.Error).Str("email", email).Msg("Database query failed during handleOAuthLogin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during login process"})
		return nil
	}

	return &user
}

func finishOAuthLogin(c *gin.Context, user *models.User) {
	// 3. Generate Token
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate token during OAuth")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	logger.Info().Str("user_id", user.ID).Msg("User logged in via OAuth")

	// 4. Redirect to frontend with token
	redirectURL := fmt.Sprintf("%s/oauth-callback?token=%s", config.AppConfig.FrontendURL, token)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// --- Forgot Password ---

type ForgotPasswordInput struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordInput struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func ForgotPassword(c *gin.Context) {
	var input ForgotPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		// Don't reveal if user exists or not
		logger.Info().Str("email", input.Email).Msg("Forgot password requested (user not found or ok)")
		c.JSON(http.StatusOK, gin.H{"message": "If this email is registered, you will receive a password reset link."})
		return
	}

	// Generate Reset Token
	resetToken := uuid.New().String()
	user.ResetToken = resetToken
	expiry := time.Now().Add(15 * time.Minute)
	user.ResetTokenExpiry = &expiry // 15 mins expiry

	if err := database.DB.Save(&user).Error; err != nil {
		logger.Error().Err(err).Msg("Failed to generate reset token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reset token"})
		return
	}

	// In a real app, send email here.
	logger.Info().Str("reset_token", resetToken).Msg("Password reset token generated")
	// Use Fprintf or similar if you really need to output to console for user to copy-paste easily without JSON clutter
	fmt.Printf("\n🔗 Reset Link: %s/auth/reset-password?token=%s\n\n", config.AppConfig.FrontendURL, resetToken)

	c.JSON(http.StatusOK, gin.H{
		"message": "If this email is registered, you will receive a password reset link.",
	})
}

func ResetPassword(c *gin.Context) {
	var input ResetPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate Password Strength
	if err := validatePasswordStrength(input.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("reset_token = ?", input.Token).First(&user).Error; err != nil {
		logger.Warn().Str("token", input.Token).Msg("Password reset failed: invalid token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		return
	}

	if user.ResetTokenExpiry != nil && time.Now().After(*user.ResetTokenExpiry) {
		logger.Warn().Str("token", input.Token).Msg("Password reset failed: expired token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token expired"})
		return
	}

	// Update Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to hash password during reset")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user.Password = string(hashedPassword)
	user.ResetToken = "" // Clear token
	// user.ResetTokenExpiry = time.Time{} // Clear expiry or leave as is since token is cleared

	if err := database.DB.Save(&user).Error; err != nil {
		logger.Error().Err(err).Msg("Failed to update password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	logger.Info().Str("user_id", user.ID).Msg("Password reset successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}
