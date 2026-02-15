package handler

import "strings"

func shouldRequireAuth() bool {
	return strings.EqualFold(strings.TrimSpace(getEnv("REQUIRE_AUTH")), "true")
}

func hasBearerToken(headers map[string]string) bool {
	auth := getHeader(headers, "authorization")
	if auth == "" {
		return false
	}
	return strings.HasPrefix(strings.ToLower(auth), "bearer ")
}

func getHeader(headers map[string]string, key string) string {
	for k, v := range headers {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}
