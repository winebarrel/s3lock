package main

import (
	"context"
	"os"

	"github.com/alecthomas/kong"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/winebarrel/s3lock/cmd/subcmd"
)

var version string

var cli struct {
	Version kong.VersionFlag
	Lock    subcmd.LockCmd   `cmd:""`
	Unlock  subcmd.UnlockCmd `cmd:""`
}

func main() {
	kctx := kong.Parse(&cli, kong.Vars{"version": version})
	cfg, err := config.LoadDefaultConfig(context.Background())
	kctx.FatalIfErrorf(err)
	err = kctx.Run(&subcmd.Context{
		Output: os.Stderr,
		S3:     s3.NewFromConfig(cfg),
	})
	kctx.FatalIfErrorf(err)
}
