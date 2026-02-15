package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	metricNameIngestion   = "AnalyticsEventIngestion"
	metricResultSuccess   = "Success"
	metricResultFailure   = "Failure"
	metricResultDuplicate = "Duplicate"
)

func logJSON(level, message, requestID, eventID string, fields map[string]interface{}) {
	payload := map[string]interface{}{
		"level":     level,
		"message":   message,
		"requestId": requestID,
		"eventId":   eventID,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	}

	for k, v := range fields {
		payload[k] = v
	}

	line, _ := json.Marshal(payload)
	fmt.Println(string(line))
}

func emitMetric(metricName, result string, value float64) {
	namespace := strings.TrimSpace(getEnv("METRICS_NAMESPACE"))
	if namespace == "" {
		namespace = "Veet/AnalyticsEvents"
	}

	metric := map[string]interface{}{
		"_aws": map[string]interface{}{
			"Timestamp": time.Now().UnixMilli(),
			"CloudWatchMetrics": []map[string]interface{}{
				{
					"Namespace":  namespace,
					"Dimensions": [][]string{{"Result"}},
					"Metrics": []map[string]string{
						{"Name": metricName, "Unit": "Count"},
					},
				},
			},
		},
		"Result": result,
	}
	metric[metricName] = value

	line, _ := json.Marshal(metric)
	fmt.Println(string(line))
}

func getEnv(name string) string {
	return os.Getenv(name)
}
