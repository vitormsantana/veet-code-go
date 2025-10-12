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

type Question struct {
    Name       string   `dynamodbav:"question_name"`
    Date       string   `dynamodbav:"question_solved_date"`
    Difficulty string   `dynamodbav:"difficulty"`
    Tags       []string `json:"tags"`
}

type DayStatistic struct {
    Date  string `json:"date"`
    Count int    `json:"count"`
}

type Statistics struct {
    QuestionsCrackedPerDay              []DayStatistic      `json:"questionsCrackedPerDay"`
    QuestionsCrackedPerDifficulty       map[string]int      `json:"questionsCrackedPerDifficulty"`
    QuestionsCrackedPerTag              map[string]int      `json:"questionsCrackedPerTag"`
    TotalQuestionsCracked               int                 `json:"totalQuestionsCracked"`
    IncrementalQuestionsCrackedPerDay   []DayStatistic      `json:"incrementalQuestionsCrackedPerDay"`
}

var dynamoClient *dynamodb.Client
const tableName = "veet_code_questions_table"

func init() {
    cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("sa-east-1"))
    if err != nil {
        log.Fatalf("Unable to load AWS SDK config: %v", err)
    }
    dynamoClient = dynamodb.NewFromConfig(cfg)
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    questions, err := fetchAllQuestions(ctx)
    if err != nil {
        log.Printf("Failed to fetch questions: %v", err)
        return events.APIGatewayProxyResponse{
            StatusCode: 500,
            Body:       "Internal Server Error",
        }, nil
    }

    stats := generateStatistics(questions)

    responseBody, err := json.Marshal(stats)
    if err != nil {
        log.Printf("Failed to marshal response: %v", err)
        return events.APIGatewayProxyResponse{
            StatusCode: 500,
            Body:       "Internal Server Error",
        }, nil
    }

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

func fetchAllQuestions(ctx context.Context) ([]Question, error) {
    var questions []Question
    input := &dynamodb.ScanInput{
        TableName: aws.String(tableName),
    }

    paginator := dynamodb.NewScanPaginator(dynamoClient, input)
    for paginator.HasMorePages() {
        page, err := paginator.NextPage(ctx)
        if err != nil {
            return nil, fmt.Errorf("failed to scan DynamoDB: %w", err)
        }

        var pageQuestions []struct {
            Name       string `dynamodbav:"question_name"`
            Date       string `dynamodbav:"question_solved_date"`
            Difficulty string `dynamodbav:"difficulty"`
            Tags       string `dynamodbav:"tags"`
        }
        err = attributevalue.UnmarshalListOfMaps(page.Items, &pageQuestions)
        if err != nil {
            return nil, fmt.Errorf("failed to unmarshal DynamoDB items: %w", err)
        }

        for _, q := range pageQuestions {
            var tags []string
            if err := json.Unmarshal([]byte(q.Tags), &tags); err != nil {
                log.Printf("Failed to parse tags for question %s: %v", q.Name, err)
                tags = []string{}
            }

            questions = append(questions, Question{
                Name:       q.Name,
                Date:       q.Date,
                Difficulty: q.Difficulty,
                Tags:       tags,
            })
        }
    }

    return questions, nil
}

func generateStatistics(questions []Question) Statistics {
    stats := Statistics{
        QuestionsCrackedPerDifficulty: make(map[string]int),
        QuestionsCrackedPerTag:        make(map[string]int),
        TotalQuestionsCracked:         0,
    }

    dailyStats := make(map[string]int)

    for _, q := range questions {
        dailyStats[q.Date]++
        stats.QuestionsCrackedPerDifficulty[q.Difficulty]++
        for _, tag := range q.Tags {
            stats.QuestionsCrackedPerTag[tag]++
        }
        stats.TotalQuestionsCracked++
    }

    sortedDates := getSortedDates(dailyStats)

    // Populate ordered statistics
    var orderedQuestions []DayStatistic
    var incrementalQuestions []DayStatistic
    runningTotal := 0
    for _, date := range sortedDates {
        count := dailyStats[date]
        orderedQuestions = append(orderedQuestions, DayStatistic{Date: date, Count: count})
        runningTotal += count
        incrementalQuestions = append(incrementalQuestions, DayStatistic{Date: date, Count: runningTotal})
    }

    stats.QuestionsCrackedPerDay = orderedQuestions
    stats.IncrementalQuestionsCrackedPerDay = incrementalQuestions

    return stats
}

func getSortedDates(dateMap map[string]int) []string {
    var dates []string
    for date := range dateMap {
        dates = append(dates, date)
    }

    sort.SliceStable(dates, func(i, j int) bool {
        layout := "02/01/2006" // Adjust the date format as per your data
        date1, err1 := time.Parse(layout, dates[i])
        date2, err2 := time.Parse(layout, dates[j])
        if err1 != nil || err2 != nil {
            log.Printf("Error parsing dates: %v, %v", err1, err2)
            return dates[i] < dates[j]
        }
        return date1.Before(date2)
    })

    return dates
}

func main() {
    lambda.Start(Handler)
}
