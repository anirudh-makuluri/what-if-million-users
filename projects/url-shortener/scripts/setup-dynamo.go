package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)


func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
	config.WithRegion("us-east-1"),
			config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("fake", "fake", ""),
		),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, opts ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: "http://localhost:8000"}, nil
				},
			),
		),
	)

	if err != nil {
		log.Fatal(err)
	}

	client := dynamodb.NewFromConfig(cfg)

	_, err = client.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		TableName:   aws.String("urls"),
		BillingMode: types.BillingModePayPerRequest,
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("short_code"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("short_code"),
				KeyType:       types.KeyTypeHash,
			},
		},
	})

	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	fmt.Println("Created table successfully")


}