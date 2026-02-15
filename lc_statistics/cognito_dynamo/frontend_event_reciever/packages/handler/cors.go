package handler

import (
	"os"
	"strings"
)

func corsHeaders(requestHeaders map[string]string) map[string]string {
	origin := getHeader(requestHeaders, "origin")
	allowedOrigin := resolveAllowedOrigin(origin)
	hasAllowlist := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS")) != ""
	if allowedOrigin == "" && !hasAllowlist {
		allowedOrigin = "*"
	}

	return map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  allowedOrigin,
		"Access-Control-Allow-Methods": "POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
		"Vary":                         "Origin",
	}
}

func resolveAllowedOrigin(origin string) string {
	allowlist := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if allowlist == "" {
		return "*"
	}

	for _, entry := range strings.Split(allowlist, ",") {
		candidate := strings.TrimSpace(entry)
		if candidate == "*" {
			return "*"
		}
		if origin != "" && candidate == origin {
			return origin
		}
	}

	return ""
}
