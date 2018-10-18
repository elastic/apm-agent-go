// +build ignore

package main

import (
	"context" // Trace lambda function invocations.

	"github.com/aws/aws-lambda-go/lambda"

	_ "go.elastic.co/apm/module/apmlambda"
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
