package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
)

type Request struct {
	QuestionName       string   `json:"name"`
	QuestionDate       string   `json:"date"`
	QuestionDifficulty string   `json:"difficulty"`
	QuestionTags       []string `json:"tags"`
}

type Response struct {
	Message string `json:"message"`
}

func Handler(ctx context.Context, request Request) (Response, error) {
	fmt.Println("Question Name: ", request.QuestionName)
	fmt.Println("Question Date: ", request.QuestionDate)
	fmt.Println("Question Difficulty: ", request.QuestionDifficulty)
	fmt.Println("Question Tags: ", request.QuestionTags)

	message := fmt.Sprintf("Question Name: %s, Question Date: %s, Question Difficulty: %s, Question Tags: %s", request.QuestionName, request.QuestionDate, request.QuestionDifficulty, request.QuestionTags)

	return Response{Message: message}, nil
}

func main() {
	lambda.Start(Handler)
}
