package subcmd

import (
	"os"

	"github.com/winebarrel/s3lock"
)

type UnlockCmd struct {
	LockFile string `arg:"" help:"Lock info file path"`
}

func (cmd *UnlockCmd) Run(cmdCtx *Context) error {
	j, err := os.ReadFile(cmd.LockFile)

	if err != nil {
		return err
	}

	lock, err := s3lock.NewLockFromJSON(cmdCtx.S3, j)

	if err != nil {
		return err
	}

	return lock.Unlock()
}
