package subcmd_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	os.Setenv("AWS_REGION", "us-east-1")

	m.Run()
}
