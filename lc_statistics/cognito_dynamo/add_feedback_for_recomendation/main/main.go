package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_feedback_for_recomendation/packages/handler"
)

func main() {
	lambda.Start(handler.Handler)
}
