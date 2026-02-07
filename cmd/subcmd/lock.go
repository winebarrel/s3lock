package subcmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/winebarrel/s3lock"
)

type LockCmd struct {
	S3URL *url.URL `arg:"" name:"s3-url" help:"S3 URL of the object to lock, e.g., s3://bucket/lock-obj-key"`
	Wait  uint     `short:"w" help:"Fail if the lock cannot be acquired within seconds"`
}

func (cmd *LockCmd) AfterApply() error {
	if cmd.S3URL.Scheme != "s3" || cmd.S3URL.Host == "" || strings.TrimPrefix(cmd.S3URL.Path, "/") == "" {
		return fmt.Errorf("invalid S3 URL: %s", cmd.S3URL)
	}

	return nil
}

func (cmd *LockCmd) Run(cmdCtx *Context) error {
	ctx := context.Background()
	lockObj := s3lock.New(cmdCtx.S3, cmd.S3URL.Host, strings.TrimPrefix(cmd.S3URL.Path, "/"))

	var lock *s3lock.Lock
	var err error

	if cmd.Wait > 0 {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(cmd.Wait)*time.Second)
		lock, err = lockObj.LockWait(ctx)
		cancel()
	} else {
		lock, err = lockObj.Lock(ctx)
	}

	if err != nil {
		return err
	}

	j, err := lock.MarshalJSON()

	if err != nil {
		return err
	}

	fmt.Fprintln(cmdCtx.Output, string(j)) //nolint:errcheck

	return nil
}
