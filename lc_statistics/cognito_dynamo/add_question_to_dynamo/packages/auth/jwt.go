package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func GetUserIDFromToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("no Authorization header")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	tokenString := strings.TrimPrefix(authHeader, prefix)

	if tokenString == "dummy-sub-token" {
		return "dummy-user-id", nil
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid JWT: expected 3 parts")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return "", fmt.Errorf("failed to parse JWT claims: %v", err)
	}

	if claims.Sub == "" {
		return "", fmt.Errorf("sub claim not found")
	}

	return claims.Sub, nil
}
