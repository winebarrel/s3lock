package subcmd

import (
	"fmt"
	"os"

	"github.com/winebarrel/s3lock"
)

type UnlockCmd struct {
	LockFile string `arg:"" help:"Lock file path."`
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

	err = lock.Unlock()

	if err != nil {
		return err
	}

	fmt.Fprintf(cmdCtx.Output, "%s has been unlocked\n", lock) //nolint:errcheck
	err = os.Remove(cmd.LockFile)

	if err != nil {
		return err
	}

	fmt.Fprintf(cmdCtx.Output, "delete %s\n", cmd.LockFile) //nolint:errcheck

	return nil
}
