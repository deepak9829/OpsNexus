package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func NewClient(endpointURL, region, accessKey, secretKey string) (*dynamodb.Client, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	// LocalStack / local dev: custom endpoint + static credentials
	if endpointURL != "" {
		resolver := aws.EndpointResolverWithOptionsFunc(func(service, reg string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: endpointURL, SigningRegion: region}, nil
		})
		opts = append(opts, config.WithEndpointResolverWithOptions(resolver))
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}
	// When endpointURL is empty (EKS/production), LoadDefaultConfig picks up
	// IRSA credentials automatically via the WebIdentity token file.

	cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	return dynamodb.NewFromConfig(cfg), nil
}
