package store

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type URLRecord struct {
	ShortCode string `dynamodbav:"short_code"`
	LongURL string `dynamodbav:"long_url"`
	UserID string `dynamodbav:"user_id"`
	CreatedAt string `dynamodbav:"created_at"`
}

type DynamoStore struct {
	client *dynamodb.Client
	tableName string
}

func NewDynamoStore() (*DynamoStore, error) {
	endpoint := os.Getenv("DYNAMO_ENDPOINT")
	region := os.Getenv("DYNAMO_REGION")
	tableName := os.Getenv("DYNAMO_TABLE")

	optFns := []func(*config.LoadOptions) error {
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("fake", "fake", ""),
		),
	}

	// only override endpoint for local development
	if endpoint != "" {
		optFns = append(optFns, config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, opts ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: endpoint}, nil
				},
			),
		))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), optFns...)
	if err != nil {
		return nil, err
	}

	return &DynamoStore{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: tableName,
	}, nil
}


func (d *DynamoStore) SaveURL(ctx context.Context, record URLRecord) error {
	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return err
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item: item,
	})
	return err
}



func (d *DynamoStore) GetURL(ctx context.Context, shortCode string) (*URLRecord, error) {
	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"short_code": &types.AttributeValueMemberS{Value: shortCode},
		},
	})
	if err != nil {
		return nil, err
	}

	// item not found
	if result.Item == nil {
		return nil, nil
	}

	var record URLRecord
	err = attributevalue.UnmarshalMap(result.Item, &record)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

var ErrShortCodeExists = errors.New("short code already exists")


func (d *DynamoStore) SaveURLIfNotExists(ctx context.Context, record URLRecord) error {
	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return err
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(d.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(short_code)"),
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return ErrShortCodeExists
		}
		return err
	}

	return nil
}

