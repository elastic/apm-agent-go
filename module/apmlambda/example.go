// +build ignore

package main

import (
	"context" // Trace lambda function invocations.

	"github.com/aws/aws-lambda-go/lambda"

	_ "github.com/elastic/apm-agent-go/module/apmlambda"
)

type Request struct {
	Name string `json:"name"`
}

func Handler(ctx context.Context, req Request) (string, error) {
	return "Hello, " + req.Name, nil
}

func main() {
	lambda.Start(Handler)
}
