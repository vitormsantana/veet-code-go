package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_question_to_dynamo/packages/handler"
)

func main() {
	lambda.Start(handler.Handler)
}
