package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/frontend_event_reciever/packages/storage"
	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/frontend_event_reciever/packages/typesandstructs"
)

var (
	storageService *storage.Service
	initErr        error
)

func init() {
	storageService, initErr = storage.NewService(
		context.Background(),
		getEnv("AWS_REGION"),
		getBucketName(),
		getEnv("EVENTS_PREFIX"),
	)
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestID := event.RequestContext.RequestID
	if requestID == "" {
		requestID = "unknown"
	}
	logJSON("info", "request received", requestID, "", map[string]interface{}{
		"path":     event.Path,
		"resource": event.Resource,
		"method":   event.HTTPMethod,
	})

	headers := corsHeaders(event.Headers)
	if strings.EqualFold(event.HTTPMethod, http.MethodOptions) {
		logJSON("info", "cors preflight handled", requestID, "", nil)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNoContent,
			Headers:    headers,
		}, nil
	}

	if !strings.EqualFold(event.HTTPMethod, http.MethodPost) {
		return jsonResponse(http.StatusMethodNotAllowed, headers, map[string]string{"message": "method not allowed"}), nil
	}

	if initErr != nil {
		logJSON("error", "storage init failed", requestID, "", map[string]interface{}{"error": initErr.Error()})
		emitMetric(metricNameIngestion, metricResultFailure, 1)
		return jsonResponse(http.StatusInternalServerError, headers, map[string]string{"message": "service unavailable"}), nil
	}

	if shouldRequireAuth() && !hasBearerToken(event.Headers) {
		logJSON("warn", "authorization required but token missing", requestID, "", nil)
		emitMetric(metricNameIngestion, metricResultFailure, 1)
		return jsonResponse(http.StatusUnauthorized, headers, map[string]string{"message": "unauthorized"}), nil
	}

	eventsToProcess, isBatch, batchID, err := parseRequestBody(event.Body)
	if err != nil {
		logJSON("warn", "invalid json body", requestID, "", map[string]interface{}{"error": err.Error()})
		emitMetric(metricNameIngestion, metricResultFailure, 1)
		return jsonResponse(http.StatusBadRequest, headers, map[string]string{"message": "invalid request body"}), nil
	}

	acceptedCount := 0
	alreadyProcessedCount := 0
	for _, analyticsEvent := range eventsToProcess {
		if err := validateEvent(analyticsEvent); err != nil {
			logJSON("warn", "validation failed", requestID, analyticsEvent.EventID, map[string]interface{}{"error": err.Error()})
			emitMetric(metricNameIngestion, metricResultFailure, 1)
			if isBatch {
				return jsonResponse(http.StatusBadRequest, headers, map[string]string{
					"status":  "rejected",
					"error":   "invalid_event",
					"eventId": analyticsEvent.EventID,
					"details": err.Error(),
				}), nil
			}
			return jsonResponse(http.StatusBadRequest, headers, map[string]string{"message": err.Error()}), nil
		}

		result, processErr := processEvent(ctx, requestID, analyticsEvent)
		if processErr != nil {
			emitMetric(metricNameIngestion, metricResultFailure, 1)
			if result == "failed_to_persist" {
				return jsonResponse(http.StatusInternalServerError, headers, map[string]string{"message": "failed to persist event"}), nil
			}
			return jsonResponse(http.StatusInternalServerError, headers, map[string]string{"message": "failed to process event"}), nil
		}

		if result == "already_processed" {
			alreadyProcessedCount++
			continue
		}
		acceptedCount++
	}

	if isBatch {
		if strings.TrimSpace(batchID) == "" {
			batchID = "unassigned"
		}
		return jsonResponse(http.StatusAccepted, headers, map[string]interface{}{
			"status":            "accepted",
			"batchId":           batchID,
			"accepted":          acceptedCount,
			"already_processed": alreadyProcessedCount,
		}), nil
	}

	analyticsEvent := eventsToProcess[0]
	if alreadyProcessedCount == 1 {
		return jsonResponse(http.StatusAccepted, headers, map[string]string{"eventId": analyticsEvent.EventID, "status": "already_processed"}), nil
	}
	return jsonResponse(http.StatusAccepted, headers, map[string]string{"eventId": analyticsEvent.EventID, "status": "accepted"}), nil
}

func parseRequestBody(body string) ([]typesandstructs.AnalyticsEvent, bool, string, error) {
	var batch typesandstructs.AnalyticsBatchRequest
	if err := json.Unmarshal([]byte(body), &batch); err != nil {
		return nil, false, "", err
	}

	if batch.Events != nil {
		if len(batch.Events) == 0 {
			return nil, true, batch.BatchID, fmt.Errorf("events must contain at least one event")
		}
		return batch.Events, true, batch.BatchID, nil
	}

	var singleEvent typesandstructs.AnalyticsEvent
	if err := json.Unmarshal([]byte(body), &singleEvent); err != nil {
		return nil, false, "", err
	}
	return []typesandstructs.AnalyticsEvent{singleEvent}, false, "", nil
}

func processEvent(ctx context.Context, requestID string, analyticsEvent typesandstructs.AnalyticsEvent) (string, error) {
	ingestedAt := time.Now().UTC()
	persistedEvent := storage.ToPersistedEvent(analyticsEvent, ingestedAt)
	objectKey := storageService.BuildObjectKey(persistedEvent)

	exists, err := storageService.Exists(ctx, objectKey)
	if err != nil {
		logJSON("error", "failed to check idempotency object", requestID, analyticsEvent.EventID, map[string]interface{}{"error": err.Error()})
		return "failed_to_check_idempotency", err
	}

	if exists {
		logJSON("info", "event already ingested", requestID, analyticsEvent.EventID, map[string]interface{}{"objectKey": objectKey})
		emitMetric(metricNameIngestion, metricResultDuplicate, 1)
		return "already_processed", nil
	}

	if err := storageService.PutEvent(ctx, objectKey, persistedEvent); err != nil {
		logJSON("error", "failed to persist event", requestID, analyticsEvent.EventID, map[string]interface{}{"error": err.Error(), "objectKey": objectKey})
		return "failed_to_persist", err
	}

	logJSON("info", "event persisted", requestID, analyticsEvent.EventID, map[string]interface{}{"objectKey": objectKey})
	emitMetric(metricNameIngestion, metricResultSuccess, 1)
	return "accepted", nil
}
