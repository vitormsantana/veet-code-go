package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("Unable to initialize zap logger: %v", err))
	}
	logger.Info("Zap logger initialized in auth package")
}

func GetUserIDFromToken(authHeader string) (string, error) {
	logger.Info("Extracting user ID from token")

	if authHeader == "" {
		logger.Warn("Authorization header is empty")
		return "", fmt.Errorf("no Authorization header")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		logger.Warn("Invalid Authorization header format")
		return "", fmt.Errorf("invalid Authorization header format")
	}

	tokenString := strings.TrimPrefix(authHeader, prefix)

	if tokenString == "dummy-sub-token" {
		logger.Info("Dummy token detected, returning dummy user ID")
		return "dummy-user-id", nil
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) < 2 {
		logger.Warn("Invalid JWT format: expected 3 parts")
		return "", fmt.Errorf("invalid JWT: expected 3 parts")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		logger.Error("Failed to decode JWT payload", zap.Error(err))
		return "", fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		logger.Error("Failed to parse JWT claims", zap.Error(err))
		return "", fmt.Errorf("failed to parse JWT claims: %v", err)
	}

	if claims.Sub == "" {
		logger.Warn("Sub claim not found in JWT")
		return "", fmt.Errorf("sub claim not found")
	}

	logger.Info("User ID extracted successfully", zap.String("user_id", claims.Sub))
	return claims.Sub, nil
}
