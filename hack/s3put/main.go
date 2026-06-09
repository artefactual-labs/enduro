package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	var (
		endpoint  = flag.String("endpoint", "", "S3 endpoint URL")
		region    = flag.String("region", "us-west-1", "S3 region")
		bucket    = flag.String("bucket", "", "S3 bucket")
		key       = flag.String("key", "", "S3 object key")
		accessKey = flag.String("access-key", "", "S3 access key")
		secretKey = flag.String("secret-key", "", "S3 secret key")
		path      = flag.String("file", "", "Path to the file to upload")
	)
	flag.Parse()

	if *endpoint == "" || *bucket == "" || *key == "" || *accessKey == "" || *secretKey == "" || *path == "" {
		fmt.Fprintln(os.Stderr, "endpoint, bucket, key, access-key, secret-key, and file are required")
		os.Exit(2)
	}

	file, err := os.Open(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(*region),
		awsconfig.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     *accessKey,
				SecretAccessKey: *secretKey,
			},
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load AWS config: %v\n", err)
		os.Exit(1)
	}

	client := s3.NewFromConfig(cfg, func(opts *s3.Options) {
		opts.BaseEndpoint = endpoint
		opts.UsePathStyle = true
		opts.Region = *region
	})

	if _, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: bucket,
		Key:    key,
		Body:   file,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "put object: %v\n", err)
		os.Exit(1)
	}
}
