package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type BucketStats struct {
	TotalSizeBytes int64
	ObjectCount    int64
}

func GetBucketStats(ctx context.Context, endpoint, bucket, accessKey, secretKey string) (BucketStats, error) {
	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String("http://" + endpoint)
	})

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})

	var stats BucketStats
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return BucketStats{}, fmt.Errorf("list objects: %w", err)
		}
		for _, obj := range page.Contents {
			if obj.Size != nil {
				stats.TotalSizeBytes += *obj.Size
			}
			stats.ObjectCount++
		}
	}

	return stats, nil
}
