package subcmd

import (
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Context struct {
	Output io.Writer
	S3     *s3.Client
}
