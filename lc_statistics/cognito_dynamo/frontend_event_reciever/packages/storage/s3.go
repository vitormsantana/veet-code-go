package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/frontend_event_reciever/packages/typesandstructs"
)

type S3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

type Service struct {
	client S3API
	bucket string
	prefix string
}

func NewService(ctx context.Context, region, bucket, prefix string) (*Service, error) {
	if strings.TrimSpace(bucket) == "" {
		return nil, fmt.Errorf("EVENTS_BUCKET is required")
	}

	cfgOptions := []func(*config.LoadOptions) error{}
	if strings.TrimSpace(region) != "" {
		cfgOptions = append(cfgOptions, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, cfgOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	return &Service{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
		prefix: strings.Trim(prefix, "/"),
	}, nil
}

func (s *Service) BuildObjectKey(event typesandstructs.PersistedAnalyticsEvent) string {
	prefix := s.prefix
	if prefix == "" {
		prefix = "analytics-events"
	}

	partitionDate := event.Year + event.Month + event.Day

	return fmt.Sprintf(
		"%s/anomesdia=%s/app=%s/eventType=%s/%s.json",
		prefix,
		partitionDate,
		sanitizePathPart(event.App),
		sanitizePathPart(event.EventType),
		event.EventID,
	)
}

func (s *Service) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
		return false, nil
	}

	errText := strings.ToLower(err.Error())
	if strings.Contains(errText, "not found") || strings.Contains(errText, "status code: 404") {
		return false, nil
	}

	return false, fmt.Errorf("head object failed: %w", err)
}

func (s *Service) PutEvent(ctx context.Context, key string, event typesandstructs.PersistedAnalyticsEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event payload failed: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(payload),
		ContentType: aws.String("application/json"),
		Metadata: map[string]string{
			"eventId":            event.EventID,
			"eventTimestamp":     event.EventTimestamp,
			"ingestionTimestamp": event.IngestionTimestamp,
		},
	})
	if err != nil {
		return fmt.Errorf("put object failed: %w", err)
	}

	return nil
}

func ToPersistedEvent(ev typesandstructs.AnalyticsEvent, ingestedAt time.Time) typesandstructs.PersistedAnalyticsEvent {
	out := typesandstructs.PersistedAnalyticsEvent{
		EventID:            ev.EventID,
		EventTimestamp:     ev.Timestamp,
		IngestionTimestamp: ingestedAt.UTC().Format(time.RFC3339Nano),
		Year:               ingestedAt.UTC().Format("2006"),
		Month:              ingestedAt.UTC().Format("01"),
		Day:                ingestedAt.UTC().Format("02"),
		App:                ev.App,
		EventType:          ev.EventType,
		Phase:              ev.Phase,
		Source:             ev.Source,
		Feature:            ev.Feature,
		Page:               ev.Page,
		Label:              ev.Label,
		Metadata:           ev.Metadata,
	}

	if ev.API != nil {
		out.APIName = ev.API.Name
		out.APIEndpoint = ev.API.Endpoint
		out.APIMethod = ev.API.Method
		out.APIStatusCode = ev.API.StatusCode
		out.APIOutcome = ev.API.Outcome
	}

	if ev.User != nil {
		out.UserSub = ev.User.Sub
		out.UserEmail = ev.User.Email
	}

	return out
}

func sanitizePathPart(v string) string {
	trimmed := strings.TrimSpace(strings.ToLower(v))
	if trimmed == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", "?", "-", "#", "-", "=", "-", "&", "-")
	return replacer.Replace(trimmed)
}
