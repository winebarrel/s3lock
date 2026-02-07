package subcmd_test

import (
	"context"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func testNewS3Client(t *testing.T) *s3.Client {
	t.Helper()
	cfg, err := config.LoadDefaultConfig(t.Context())

	if err != nil {
		t.Fatal(err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		// for S3Mock: https://github.com/adobe/S3Mock#path-style-vs-domain-style-access
		o.UsePathStyle = true
	})

	return client
}

func testDeleteObject(t *testing.T, client *s3.Client, bucket string, key string) {
	t.Helper()
	input := &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}
	client.DeleteObject(context.Background(), input)
}

func testGetObject(t *testing.T, client *s3.Client, bucket string, key string) (string, error) {
	t.Helper()
	input := &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}
	output, err := client.GetObject(context.Background(), input)

	if err != nil {
		return "", err
	}

	defer output.Body.Close()

	b, err := io.ReadAll(output.Body)

	if err != nil {
		t.Fatal(err)
	}

	return string(b), nil
}
