package subcmd_test

import (
	"bytes"
	"io"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/winebarrel/s3lock"
	"github.com/winebarrel/s3lock/cmd/subcmd"
)

func TestLockCmd(t *testing.T) {
	s3cli := testNewS3Client(t)
	testDeleteObject(t, s3cli, "s3lock-test", "lock-obj")

	cmd := &subcmd.LockCmd{
		S3URL: &url.URL{Scheme: "s3", Host: "s3lock-test", Path: "/lock-obj"},
	}

	var buf bytes.Buffer

	err := cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: &buf,
	})

	require.NoError(t, err)
	require.Regexp(t, `{"Bucket":"s3lock-test","Key":"lock-obj","Id":"\w{8}-\w{4}-\w{4}-\w{4}-\w{12}","ETag":"\\"\w{32}\\""}`, buf.String())

	body, err := testGetObject(t, s3cli, "s3lock-test", "lock-obj")
	require.NoError(t, err)
	require.Regexp(t, `\w{8}-\w{4}-\w{4}-\w{4}-\w{12}`, body)
}

func TestLockCmdWithWait(t *testing.T) {
	s3cli := testNewS3Client(t)
	testDeleteObject(t, s3cli, "s3lock-test", "lock-obj")

	cmd := &subcmd.LockCmd{
		S3URL: &url.URL{Scheme: "s3", Host: "s3lock-test", Path: "/lock-obj"},
		Wait:  1,
	}

	var buf bytes.Buffer

	err := cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: &buf,
	})

	require.NoError(t, err)
	require.Regexp(t, `{"Bucket":"s3lock-test","Key":"lock-obj","Id":"\w{8}-\w{4}-\w{4}-\w{4}-\w{12}","ETag":"\\"\w{32}\\""}`, buf.String())

	body, err := testGetObject(t, s3cli, "s3lock-test", "lock-obj")
	require.NoError(t, err)
	require.Regexp(t, `\w{8}-\w{4}-\w{4}-\w{4}-\w{12}`, body)
}

func TestLockCmdLockAlreadyHeld(t *testing.T) {
	s3cli := testNewS3Client(t)
	testDeleteObject(t, s3cli, "s3lock-test", "lock-obj")

	cmd := &subcmd.LockCmd{
		S3URL: &url.URL{Scheme: "s3", Host: "s3lock-test", Path: "/lock-obj"},
	}

	err := cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: io.Discard,
	})

	require.NoError(t, err)

	err = cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: io.Discard,
	})

	require.ErrorIs(t, err, s3lock.ErrLockAlreadyHeld)
}

func TestLockCmdFatal(t *testing.T) {
	s3cli := testNewS3Client(t)
	testDeleteObject(t, s3cli, "s3lock-test", "lock-obj")

	cmd := &subcmd.LockCmd{
		S3URL: &url.URL{Scheme: "s3", Host: "xxx-s3lock-test", Path: "/lock-obj"},
	}

	err := cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: io.Discard,
	})

	require.ErrorContains(t, err, "The specified bucket does not exist")
}
