package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

const tableName = "studies_table"

var dynamoClient *dynamodb.Client

type StudyRecord struct {
	Date    string `json:"date" dynamodbav:"study_date"`
	Theme   string `json:"theme" dynamodbav:"study_theme"`
	Minutes int    `json:"minutes" dynamodbav:"minutes_of_study"`
}

type DayStatistic struct {
	Date    string         `json:"date"`
	Minutes int            `json:"minutes"`
	Themes  map[string]int `json:"themes"`
}

type Statistics struct {
	TotalMinutesStudied   int                          `json:"totalMinutesStudied"`
	TotalMinutesPerDay    []DayStatistic               `json:"totalMinutesPerDay"`
	MinutesPerThemePerDay map[string]map[string]int    `json:"minutesPerThemePerDay"`
}

func init() {
	// Initialize DynamoDB client
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("sa-east-1"))
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
}

// Handler processes the incoming event and returns the statistics
func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Fetch study records from DynamoDB
	records, err := fetchStudyRecords(ctx)
	if err != nil {
		log.Printf("Failed to fetch records: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	// Generate statistics from records
	stats := generateStatistics(records)

	// Marshal statistics into JSON response
	responseBody, err := json.Marshal(stats)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	// Return the API response
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                   "application/json",
			"Access-Control-Allow-Origin":    "*",
			"Access-Control-Allow-Methods":   "GET, OPTIONS",
			"Access-Control-Allow-Headers":   "Content-Type, Authorization",
		},
		Body: string(responseBody),
	}, nil
}

// fetchStudyRecords scans DynamoDB and returns a list of StudyRecord
func fetchStudyRecords(ctx context.Context) ([]StudyRecord, error) {
	var records []StudyRecord
	input := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}

	paginator := dynamodb.NewScanPaginator(dynamoClient, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan DynamoDB: %w", err)
		}

		var pageRecords []StudyRecord
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageRecords)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal DynamoDB items: %w", err)
		}

		records = append(records, pageRecords...)
	}

	return records, nil
}

// generateStatistics processes the study records and calculates statistics
func generateStatistics(records []StudyRecord) Statistics {
	// Sort records by date
	sort.Slice(records, func(i, j int) bool {
		dateI, _ := time.Parse("02/01/2006", records[i].Date)
		dateJ, _ := time.Parse("02/01/2006", records[j].Date)
		return dateI.Before(dateJ)
	})

	// Prepare data structures for statistics
	themeMinutes := make(map[string]int)
	minutesPerThemePerDay := make(map[string]map[string]int)
	totalMinutesPerDay := []DayStatistic{}
	totalMinutesStudied := 0

	// Process records to generate statistics
	for _, record := range records {
		// Ensure theme is initialized in the minutes per theme per day map
		if _, ok := minutesPerThemePerDay[record.Theme]; !ok {
			minutesPerThemePerDay[record.Theme] = make(map[string]int)
		}

		// Update cumulative total for the theme
		themeMinutes[record.Theme] += record.Minutes
		minutesPerThemePerDay[record.Theme][record.Date] = themeMinutes[record.Theme]

		// Add to total minutes for the day or create a new entry
		addToTotalMinutesPerDay(&totalMinutesPerDay, record)

		// Update global total minutes studied
		totalMinutesStudied += record.Minutes
	}

	// Return the statistics
	return Statistics{
		TotalMinutesStudied:   totalMinutesStudied,
		TotalMinutesPerDay:    totalMinutesPerDay,
		MinutesPerThemePerDay: minutesPerThemePerDay,
	}
}

func addToTotalMinutesPerDay(totalMinutesPerDay *[]DayStatistic, record StudyRecord) {
	// Check if the date already exists in the totalMinutesPerDay slice
	var found bool
	for i := range *totalMinutesPerDay {
		if (*totalMinutesPerDay)[i].Date == record.Date {
			// Add the current day's minutes to the total of the previous day
			(*totalMinutesPerDay)[i].Minutes += record.Minutes
			(*totalMinutesPerDay)[i].Themes[record.Theme] += record.Minutes
			found = true
			break
		}
	}

	// If not found, create a new DayStatistic entry
	if !found {
		// If there's a previous day, add its minutes to the current day's total
		var previousDayMinutes int
		if len(*totalMinutesPerDay) > 0 {
			previousDayMinutes = (*totalMinutesPerDay)[len(*totalMinutesPerDay)-1].Minutes
		}

		// Add the current day's minutes to the previous day's total
		*totalMinutesPerDay = append(*totalMinutesPerDay, DayStatistic{
			Date:    record.Date,
			Minutes: previousDayMinutes + record.Minutes,
			Themes:  map[string]int{record.Theme: record.Minutes},
		})
	}
}


func main() {
	lambda.Start(Handler)
}
