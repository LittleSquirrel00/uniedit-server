package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT token claims.
type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret             string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
}

// DefaultJWTConfig returns default JWT configuration.
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "uniedit",
	}
}

// JWTManager handles JWT token operations.
type JWTManager struct {
	config *JWTConfig
}

// NewJWTManager creates a new JWT manager.
func NewJWTManager(config *JWTConfig) *JWTManager {
	if config == nil {
		config = DefaultJWTConfig()
	}
	return &JWTManager{config: config}
}

// GenerateAccessToken generates a new access token for the user.
func (m *JWTManager) GenerateAccessToken(user *User) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.config.AccessTokenExpiry)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
		UserID: user.ID,
		Email:  user.Email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(m.config.Secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return signedToken, expiresAt, nil
}

// GenerateRefreshToken generates a new refresh token.
func (m *JWTManager) GenerateRefreshToken() (string, string, time.Time, error) {
	// Generate a random token
	tokenID := uuid.New().String()
	rawToken := fmt.Sprintf("%s.%d", tokenID, time.Now().UnixNano())

	// Hash the token for storage
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	expiresAt := time.Now().Add(m.config.RefreshTokenExpiry)

	return rawToken, tokenHash, expiresAt, nil
}

// ValidateAccessToken validates an access token and returns the claims.
func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidTokenClaims
	}

	return claims, nil
}

// HashRefreshToken hashes a refresh token for storage/comparison.
func (m *JWTManager) HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GetAccessTokenExpiry returns the access token expiry duration.
func (m *JWTManager) GetAccessTokenExpiry() time.Duration {
	return m.config.AccessTokenExpiry
}

// GetRefreshTokenExpiry returns the refresh token expiry duration.
func (m *JWTManager) GetRefreshTokenExpiry() time.Duration {
	return m.config.RefreshTokenExpiry
}
