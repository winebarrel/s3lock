package subcmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/winebarrel/s3lock"
)

type UnlockCmd struct {
	LockFile string `arg:"" help:"Lock file path."`
	Force    bool   `short:"f" help:"Ignore unlocked errors and lock mismatch errors"`
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
		if !cmd.Force || !errors.Is(err, s3lock.ErrAlreadyUnlocked) && !errors.Is(err, s3lock.ErrLockMismatch) {
			return err
		}
	}

	fmt.Fprintf(cmdCtx.Output, "%s has been unlocked\n", lock) //nolint:errcheck
	err = os.Remove(cmd.LockFile)

	if err != nil {
		return err
	}

	fmt.Fprintf(cmdCtx.Output, "delete %s\n", cmd.LockFile) //nolint:errcheck

	return nil
}
