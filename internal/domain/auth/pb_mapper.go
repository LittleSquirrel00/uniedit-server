package auth

import (
	"time"

	authv1 "github.com/uniedit/server/api/pb/auth"
	"github.com/uniedit/server/internal/model"
)

func toTokenPairPB(t *model.TokenPair) *authv1.TokenPairResponse {
	if t == nil {
		return nil
	}
	return &authv1.TokenPairResponse{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		ExpiresIn:    t.ExpiresIn,
		ExpiresAt:    t.ExpiresAt.UTC().Format(time.RFC3339Nano),
	}
}

func toUserAPIKeyPB(k *model.UserAPIKey) *authv1.UserAPIKey {
	if k == nil {
		return nil
	}
	lastUsedAt := ""
	if k.LastUsedAt != nil {
		lastUsedAt = k.LastUsedAt.UTC().Format(time.RFC3339Nano)
	}
	return &authv1.UserAPIKey{
		Id:         k.ID.String(),
		Provider:   k.Provider,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		Scopes:     []string(k.Scopes),
		LastUsedAt: lastUsedAt,
		CreatedAt:  k.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toSystemAPIKeyPB(k *model.SystemAPIKey) *authv1.SystemAPIKey {
	if k == nil {
		return nil
	}

	lastUsedAt := ""
	if k.LastUsedAt != nil {
		lastUsedAt = k.LastUsedAt.UTC().Format(time.RFC3339Nano)
	}
	expiresAt := ""
	if k.ExpiresAt != nil {
		expiresAt = k.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	lastRotatedAt := ""
	if k.LastRotatedAt != nil {
		lastRotatedAt = k.LastRotatedAt.UTC().Format(time.RFC3339Nano)
	}

	rotateAfterDays := int32(0)
	if k.RotateAfterDays != nil {
		rotateAfterDays = int32(*k.RotateAfterDays)
	}

	return &authv1.SystemAPIKey{
		Id:                k.ID.String(),
		Name:              k.Name,
		KeyPrefix:         k.KeyPrefix,
		Scopes:            []string(k.Scopes),
		RateLimitRpm:      int32(k.RateLimitRPM),
		RateLimitTpm:      int32(k.RateLimitTPM),
		TotalRequests:     k.TotalRequests,
		TotalInputTokens:  k.TotalInputTokens,
		TotalOutputTokens: k.TotalOutputTokens,
		TotalCostUsd:      k.TotalCostUSD,
		CacheHits:         k.CacheHits,
		CacheMisses:       k.CacheMisses,
		IsActive:          k.IsActive,
		LastUsedAt:        lastUsedAt,
		ExpiresAt:         expiresAt,
		AllowedIps:        []string(k.AllowedIPs),
		RotateAfterDays:   rotateAfterDays,
		LastRotatedAt:     lastRotatedAt,
		CreatedAt:         k.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

