package main

import (
	"archive/zip"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

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
		keyPrefix = flag.String("key-prefix", "transfer", "S3 object key prefix used for generated multi-upload keys")
		accessKey = flag.String("access-key", "", "S3 access key")
		secretKey = flag.String("secret-key", "", "S3 secret key")
		path      = flag.String("file", "", "Path to the file to upload")
		count     = flag.Int("count", 1, "Number of objects to upload")
		generate  = flag.Bool("generate-transfer", false, "Generate a small zipped transfer instead of reading -file")
	)
	flag.Parse()

	if *endpoint == "" || *bucket == "" || *accessKey == "" || *secretKey == "" {
		fmt.Fprintln(os.Stderr, "endpoint, bucket, access-key, and secret-key are required")
		os.Exit(2)
	}
	if *count < 1 {
		fmt.Fprintln(os.Stderr, "count must be greater than zero")
		os.Exit(1)
	}
	if *generate && *path != "" {
		fmt.Fprintln(os.Stderr, "generate-transfer and file cannot be used together")
		os.Exit(2)
	}
	if !*generate && *path == "" {
		fmt.Fprintln(os.Stderr, "file is required unless generate-transfer is set")
		os.Exit(2)
	}
	if *count == 1 && *key == "" && *keyPrefix == "" {
		fmt.Fprintln(os.Stderr, "key or key-prefix is required")
		os.Exit(2)
	}

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

	for i := 1; i <= *count; i++ {
		objectKey, err := nextKey(*key, *keyPrefix, *count, i)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}

		err = putTransfer(ctx, client, *bucket, objectKey, *path, *generate, i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "put object %q: %v\n", objectKey, err)
			os.Exit(1)
		}
		fmt.Printf("uploaded s3://%s/%s\n", *bucket, objectKey)
	}
}

func putTransfer(ctx context.Context, client *s3.Client, bucket, key, path string, generate bool, index int) error {
	if generate {
		payload, err := generateTransfer(key, index)
		if err != nil {
			return fmt.Errorf("generate transfer: %w", err)
		}
		return putObject(ctx, client, bucket, key, bytes.NewReader(payload), int64(len(payload)))
	}

	return putFile(ctx, client, bucket, key, path)
}

func nextKey(key, prefix string, count, index int) (string, error) {
	if count == 1 && key != "" {
		return key, nil
	}
	if prefix == "" {
		return "", fmt.Errorf("key-prefix is required when uploading multiple objects or when key is empty")
	}

	return fmt.Sprintf("%s-%03d.zip", strings.TrimSuffix(prefix, ".zip"), index), nil
}

func putFile(ctx context.Context, client *s3.Client, bucket, key, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	return putObject(ctx, client, bucket, key, file, stat.Size())
}

func putObject(ctx context.Context, client *s3.Client, bucket, key string, body io.Reader, size int64) error {
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
	})
	return err
}

func generateTransfer(key string, index int) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	files := map[string]string{
		"objects/hello.txt": fmt.Sprintf("Generated transfer %03d for %s\n", index, key),
		"metadata/source.txt": fmt.Sprintf(
			"Created by hack/s3put at %s\nObject key: %s\n",
			time.Now().UTC().Format(time.RFC3339),
			key,
		),
	}

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(w, content); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
