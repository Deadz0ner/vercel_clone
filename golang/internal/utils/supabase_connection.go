package utils

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"vercel-clone/internal/config"
)

func NewSupabaseS3Client(ctx context.Context) (*s3.Client, error) {
	cfgValues := config.Load()
	region := cfgValues.SupabaseS3Region
	endpoint := cfgValues.SupabaseS3Endpoint
	accessKeyID := cfgValues.SupabaseS3AccessKeyID
	secretAccessKey := cfgValues.SupabaseS3SecretAccessKey

	cfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(endpoint)
	})

	if client == nil {
		return nil, fmt.Errorf("failed to create s3 client from config, region=%s, endpoint=%s, accessKeyId=%s", region, endpoint, accessKeyID)
	}

	return client, nil
}
