package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/frontend_event_reciever/packages/handler"
)

func main() {
	lambda.Start(handler.Handler)
}
