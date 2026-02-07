package s3lock_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	os.Setenv("AWS_DEFAULT_REGION", "us-east-1")
	os.Setenv("AWS_ENDPOINT_URL", "http://localhost:9090")

	m.Run()
}
