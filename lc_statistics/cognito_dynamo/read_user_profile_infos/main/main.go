package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_user_profile_infos/packages/handler"
)

func main() {
	lambda.Start(handler.Handler)
}
