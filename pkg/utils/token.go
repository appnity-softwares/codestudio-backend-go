package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/config"
)

type Claims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

// GetJTI returns the token's unique ID for blacklist operations
func (c *Claims) GetJTI() string {
	return c.ID
}

// GetExpiresAt returns the token's expiration time
func (c *Claims) GetExpiresAt() time.Time {
	if c.ExpiresAt != nil {
		return c.ExpiresAt.Time
	}
	return time.Time{}
}

func GenerateToken(userID string) (string, error) {
	expirationTime := time.Now().Add(7 * 24 * time.Hour) // 7 days

	// P0 FIX: Add unique jti for token revocation support
	jti := uuid.New().String()

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti, // P0 FIX: JWT ID for blacklist
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "devconnect-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.AppConfig.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// Helper for IDs
func GenerateID() string {
	return uuid.New().String()
}
