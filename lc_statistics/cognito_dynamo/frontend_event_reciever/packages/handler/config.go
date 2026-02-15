package handler

import "strings"

func getBucketName() string {
	if v := strings.TrimSpace(getEnv("events_bucket_name")); v != "" {
		return v
	}
	return strings.TrimSpace(getEnv("EVENTS_BUCKET"))
}
