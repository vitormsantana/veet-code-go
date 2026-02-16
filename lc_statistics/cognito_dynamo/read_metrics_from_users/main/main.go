package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/read_metrics_from_users/packages/handler"
)

func main() {
	lambda.Start(handler.Handler)
}
