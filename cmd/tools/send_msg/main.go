package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func main() {
	errorFlag := flag.Bool("error", false, "Send message with error")
	flag.Parse()

	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     "test",
				SecretAccessKey: "test",
			}, nil
		})),
	)
	if err != nil {
		panic(err)
	}

	cfg.BaseEndpoint = aws.String("http://127.0.0.1:4566")

	client := sqs.NewFromConfig(cfg)
	// queueUrl := "http://localhost:4566/000000000000/gofibercorequeue"
	queueUrl := "http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/gofibercorequeue"

	body := `{
		"process_type_id": 5,
		"sede_id": 1,
		"override_process_version_id": 8,
		"roadmap": 0,
		"input": {
			"autoInvoke": true,
			"last_id_processed": 0,
			"is_last_batch": false
		}
	}`

	if *errorFlag {
		fmt.Println("⚠️  Generating ERROR message...")
		body = `{
			"process_type_id": 9999,
			"sede_id": 1,
			"override_process_version_id": 9999,
			"roadmap": 0,
			"input": {
				"force_error": true
			}
		}`
	}

	resp, err := client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &queueUrl,
		MessageBody: &body,
	})
	if err != nil {
		fmt.Printf("Error sending message: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Message sent successfully. MessageId: %s\n", *resp.MessageId)
}
