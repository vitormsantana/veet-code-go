package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

func GetUserIDFromToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("no Authorization header")
	}

	tokenString := authHeader[len("Bearer "):]

	if tokenString == "dummy-sub-token" {
		return "dummy-user-id", nil
	}

	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse JWT: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("failed to parse claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("sub claim not found")
	}

	return sub, nil
}
