package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/read_metrics_from_users/packages/auth"
	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/read_metrics_from_users/packages/db"
	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/read_metrics_from_users/packages/observability"
	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/read_metrics_from_users/packages/structstypes"
)

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	start := time.Now()
	obs := observability.New(ctx, event, "read_metrics_from_users")
	obs.Info("request_received", nil)
	ctx, span := obs.StartSpan(ctx, "handler", nil)
	defer span.End()
	correlationID := obs.CorrelationID()

	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization, x-correlation-id, x-env, x-app-version",
		"x-correlation-id":             correlationID,
	}

	if event.HTTPMethod == http.MethodOptions {
		response := events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent, Headers: headers}
		obs.EmitMetric("Invocation", "Count", 1, map[string]string{"Result": "preflight"})
		obs.EmitMetric("LatencyMs", "Milliseconds", float64(time.Since(start).Milliseconds()), map[string]string{"Result": "preflight"})
		obs.Info("request_completed", map[string]interface{}{"status": response.StatusCode, "result": "preflight", "latency_ms": time.Since(start).Milliseconds()})
		span.SetAttr("http.status_code", response.StatusCode)
		span.SetAttr("result", "preflight")
		return response, nil
	}

	// Stage: auth
	ctx, authSpan := obs.StartSpan(ctx, "auth.parse_token", map[string]interface{}{
		"stage": "auth",
	})
	authHeader := event.Headers["Authorization"]
	userID, err := auth.GetUserIDFromToken(authHeader)
	authMs := time.Since(start).Milliseconds()
	authSpan.SetAttr("stage.latency_ms", authMs)
	if err != nil {
		obs.Warn("unauthorized", map[string]interface{}{"error": err.Error()})
		authSpan.SetStatus("ERROR", "unauthorized")
		authSpan.End()
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Headers:    headers,
			Body:       `{"message":"Unauthorized"}`,
		}
		obs.EmitMetric("Invocation", "Count", 1, map[string]string{"Result": "unauthorized"})
		obs.EmitMetric("LatencyMs", "Milliseconds", float64(time.Since(start).Milliseconds()), map[string]string{"Result": "unauthorized"})
		obs.Info("request_completed", map[string]interface{}{"status": response.StatusCode, "result": "unauthorized", "latency_ms": time.Since(start).Milliseconds()})
		span.SetStatus("ERROR", "unauthorized")
		span.SetAttr("http.status_code", response.StatusCode)
		span.SetAttr("result", "unauthorized")
		return response, nil
	}
	authSpan.End()

	// Stage: DynamoDB query
	dbStart := time.Now()
	ctx, dbSpan := obs.StartSpan(ctx, "dynamodb.fetch_metrics", map[string]interface{}{
		"stage":        "dynamodb",
		"db.system":    "dynamodb",
		"db.operation": "Query",
		"db.table":     "hammocker_user_metrics_table",
	})
	questions, err := db.FetchMetrics(ctx, userID)
	dbMs := time.Since(dbStart).Milliseconds()
	dbSpan.SetAttr("stage.latency_ms", dbMs)
	obs.EmitMetric("dynamodb_fetch_latency_ms", "Milliseconds", float64(dbMs), map[string]string{"result": "success"})
	obs.EmitMetric("dynamodb_fetch_latency_ms_last", "Milliseconds", float64(dbMs), map[string]string{"result": "success"})
	if err != nil {
		obs.Error("dynamodb_fetch_failed", map[string]interface{}{"error": err.Error()})
		dbSpan.SetStatus("ERROR", "dynamodb_fetch_failed")
		dbSpan.SetAttr("result", "error")
		dbSpan.End()
		obs.EmitMetric("dynamodb_fetch_errors", "Count", 1, map[string]string{})
		obs.EmitMetric("dynamodb_fetch_latency_ms", "Milliseconds", float64(dbMs), map[string]string{"result": "error"})
		obs.EmitMetric("dynamodb_fetch_latency_ms_last", "Milliseconds", float64(dbMs), map[string]string{"result": "error"})
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"Internal Server Error"}`,
		}
		obs.EmitMetric("Invocation", "Count", 1, map[string]string{"Result": "error"})
		obs.EmitMetric("LatencyMs", "Milliseconds", float64(time.Since(start).Milliseconds()), map[string]string{"Result": "error"})
		obs.Info("request_completed", map[string]interface{}{"status": response.StatusCode, "result": "error", "latency_ms": time.Since(start).Milliseconds()})
		span.SetStatus("ERROR", "dynamodb_fetch_failed")
		span.SetAttr("http.status_code", response.StatusCode)
		span.SetAttr("result", "error")
		return response, nil
	}
	dbSpan.SetAttr("items.count", len(questions))
	dbSpan.SetAttr("result", "success")
	dbSpan.End()
	obs.EmitMetric("MetricsFetched", "Count", float64(len(questions)), map[string]string{})
	obs.EmitMetric("dynamodb_items_fetched", "Count", float64(len(questions)), map[string]string{})
	// Local dashboard friendliness: show last fetched count even when traffic is sparse.
	obs.EmitMetric("dynamodb_items_fetched_last", "Count", float64(len(questions)), map[string]string{})

	// Stage: JSON marshal
	marshalStart := time.Now()
	ctx, marshalSpan := obs.StartSpan(ctx, "json.marshal_response", map[string]interface{}{
		"stage": "processing",
	})
	// Ensure empty results render as [] (not null), which is easier to reason about in clients/dashboards.
	if questions == nil {
		questions = []structstypes.UserMetrics{}
	}
	responseBody, err := json.Marshal(questions)
	marshalMs := time.Since(marshalStart).Milliseconds()
	marshalSpan.SetAttr("stage.latency_ms", marshalMs)
	obs.EmitMetric("json_marshal_latency_ms", "Milliseconds", float64(marshalMs), map[string]string{})
	obs.EmitMetric("json_marshal_latency_ms_last", "Milliseconds", float64(marshalMs), map[string]string{})
	if err != nil {
		obs.Error("response_marshal_failed", map[string]interface{}{"error": err.Error()})
		marshalSpan.SetStatus("ERROR", "response_marshal_failed")
		marshalSpan.End()
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"Internal Server Error"}`,
		}
		obs.EmitMetric("Invocation", "Count", 1, map[string]string{"Result": "error"})
		obs.EmitMetric("LatencyMs", "Milliseconds", float64(time.Since(start).Milliseconds()), map[string]string{"Result": "error"})
		obs.Info("request_completed", map[string]interface{}{"status": response.StatusCode, "result": "error", "latency_ms": time.Since(start).Milliseconds()})
		span.SetStatus("ERROR", "response_marshal_failed")
		span.SetAttr("http.status_code", response.StatusCode)
		span.SetAttr("result", "error")
		return response, nil
	}
	marshalSpan.End()

	response := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       string(responseBody),
	}
	obs.EmitMetric("Invocation", "Count", 1, map[string]string{"Result": "success"})
	obs.EmitMetric("LatencyMs", "Milliseconds", float64(time.Since(start).Milliseconds()), map[string]string{"Result": "success"})
	obs.Info("request_completed", map[string]interface{}{"status": response.StatusCode, "result": "success", "latency_ms": time.Since(start).Milliseconds()})
	span.SetAttr("http.status_code", response.StatusCode)
	span.SetAttr("result", "success")
	return response, nil
}
