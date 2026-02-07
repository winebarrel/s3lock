package subcmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/winebarrel/s3lock"
	"github.com/winebarrel/s3lock/cmd/subcmd"
)

func TestUnlockCmd(t *testing.T) {
	s3cli := testNewS3Client(t)
	testDeleteObject(t, s3cli, "s3lock-test", "lock-obj")

	obj := s3lock.New(s3cli, "s3lock-test", "lock-obj")
	lock, err := obj.Lock(t.Context())
	require.NoError(t, err)

	j, err := lock.MarshalJSON()
	require.NoError(t, err)
	lockInfo := filepath.Join(t.TempDir(), "lock.info")
	err = os.WriteFile(lockInfo, j, 0600)
	require.NoError(t, err)

	cmd := &subcmd.UnlockCmd{
		LockFile: lockInfo,
	}

	var buf bytes.Buffer

	err = cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: &buf,
	})

	require.NoError(t, err)
	require.Empty(t, buf.String())

	_, err = testGetObject(t, s3cli, "s3lock-test", "lock-obj")
	require.ErrorContains(t, err, "The specified key does not exist")
}

func TestUnlockCmdAlreadyUnlocked(t *testing.T) {
	s3cli := testNewS3Client(t)
	testDeleteObject(t, s3cli, "s3lock-test", "lock-obj")

	lockInfo := filepath.Join(t.TempDir(), "lock.info")
	err := os.WriteFile(lockInfo, []byte(`{"Bucket":"s3lock-test","Key":"lock-obj","Id":"my-id","ETag":"\"my-etag\""}`), 0600)
	require.NoError(t, err)

	cmd := &subcmd.UnlockCmd{
		LockFile: lockInfo,
	}

	var buf bytes.Buffer

	err = cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: &buf,
	})

	require.ErrorIs(t, err, s3lock.ErrAlreadyUnlocked)
}
