package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

func jsonResponse(statusCode int, headers map[string]string, payload interface{}) events.APIGatewayProxyResponse {
	body, err := json.Marshal(payload)
	if err != nil {
		body = []byte(`{"message":"internal server error"}`)
		statusCode = http.StatusInternalServerError
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       string(body),
	}
}
