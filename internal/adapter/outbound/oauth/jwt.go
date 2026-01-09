package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/port/outbound"
)

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret             string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

// DefaultJWTConfig returns default JWT configuration.
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
}

// jwtManager implements outbound.JWTPort.
type jwtManager struct {
	secret             []byte
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewJWTManager creates a new JWT manager.
func NewJWTManager(cfg *JWTConfig) outbound.JWTPort {
	if cfg == nil {
		cfg = DefaultJWTConfig()
	}
	return &jwtManager{
		secret:             []byte(cfg.Secret),
		accessTokenExpiry:  cfg.AccessTokenExpiry,
		refreshTokenExpiry: cfg.RefreshTokenExpiry,
	}
}

// GenerateAccessToken generates an access token.
func (m *jwtManager) GenerateAccessToken(userID uuid.UUID, email string) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.accessTokenExpiry)

	claims := jwt.MapClaims{
		"sub":   userID.String(),
		"email": email,
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}

	return signedToken, expiresAt, nil
}

// GenerateRefreshToken generates a refresh token.
func (m *jwtManager) GenerateRefreshToken() (rawToken string, tokenHash string, expiresAt time.Time, err error) {
	// Generate random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate random: %w", err)
	}

	rawToken = hex.EncodeToString(bytes)
	tokenHash = m.HashRefreshToken(rawToken)
	expiresAt = time.Now().Add(m.refreshTokenExpiry)

	return rawToken, tokenHash, expiresAt, nil
}

// ValidateAccessToken validates an access token.
func (m *jwtManager) ValidateAccessToken(tokenString string) (*outbound.JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)

	userID, err := uuid.Parse(sub)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	return &outbound.JWTClaims{
		UserID: userID,
		Email:  email,
	}, nil
}

// HashRefreshToken hashes a refresh token.
func (m *jwtManager) HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GetAccessTokenExpiry returns access token expiry duration.
func (m *jwtManager) GetAccessTokenExpiry() time.Duration {
	return m.accessTokenExpiry
}

// GetRefreshTokenExpiry returns refresh token expiry duration.
func (m *jwtManager) GetRefreshTokenExpiry() time.Duration {
	return m.refreshTokenExpiry
}

// Compile-time check
var _ outbound.JWTPort = (*jwtManager)(nil)
