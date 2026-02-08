package subcmd_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
	"github.com/winebarrel/s3lock"
	"github.com/winebarrel/s3lock/cmd/subcmd"
)

func TestUnlockCmd(t *testing.T) {
	hc := &http.Client{}
	httpmock.ActivateNonDefault(hc)
	t.Cleanup(func() { httpmock.DeactivateNonDefault(hc) })

	cfg, _ := config.LoadDefaultConfig(t.Context(), config.WithHTTPClient(hc))
	s3cli := s3.NewFromConfig(cfg)

	lockInfo := filepath.Join(t.TempDir(), "lock.info")
	err := os.WriteFile(lockInfo, []byte(`{"Bucket":"s3lock-test","Key":"lock-obj","Id":"my-id","ETag":"\"my-etag\""}`), 0600)
	require.NoError(t, err)

	cmd := &subcmd.UnlockCmd{
		LockFile: lockInfo,
	}

	var buf bytes.Buffer

	httpmock.RegisterResponder(http.MethodGet, "https://s3lock-test.s3.us-east-1.amazonaws.com/lock-obj?x-id=GetObject", func(req *http.Request) (*http.Response, error) {
		require.Equal(t, `"my-etag"`, req.Header.Get("If-Match"))
		resp := httpmock.NewStringResponse(http.StatusOK, "my-id")
		resp.Header.Set("x-amz-checksum-sha256", "kVQwchJF67pQ4+bz5XSdYrDB7HYv8636AALA6FQ3HFg=")
		resp.Header.Set("x-amz-checksum-algorithm", "sha256")
		return resp, nil
	})

	httpmock.RegisterResponder(http.MethodDelete, "https://s3lock-test.s3.us-east-1.amazonaws.com/lock-obj?x-id=DeleteObject", func(req *http.Request) (*http.Response, error) {
		require.Equal(t, `"my-etag"`, req.Header.Get("If-Match"))
		return httpmock.NewStringResponse(http.StatusOK, ""), nil
	})

	err = cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: &buf,
	})

	require.NoError(t, err)
	require.Empty(t, buf.String())
}

func TestUnlockCmdAlreadyUnlocked(t *testing.T) {
	hc := &http.Client{}
	httpmock.ActivateNonDefault(hc)
	t.Cleanup(func() { httpmock.DeactivateNonDefault(hc) })

	cfg, _ := config.LoadDefaultConfig(t.Context(), config.WithHTTPClient(hc))
	s3cli := s3.NewFromConfig(cfg)

	lockInfo := filepath.Join(t.TempDir(), "lock.info")
	err := os.WriteFile(lockInfo, []byte(`{"Bucket":"s3lock-test","Key":"lock-obj","Id":"my-id","ETag":"\"my-etag\""}`), 0600)
	require.NoError(t, err)

	cmd := &subcmd.UnlockCmd{
		LockFile: lockInfo,
	}

	httpmock.RegisterResponder(http.MethodGet, "https://s3lock-test.s3.us-east-1.amazonaws.com/lock-obj?x-id=GetObject", func(req *http.Request) (*http.Response, error) {
		require.Equal(t, `"my-etag"`, req.Header.Get("If-Match"))
		return httpmock.NewStringResponse(http.StatusNotFound, ""), nil
	})

	err = cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: io.Discard,
	})

	require.ErrorIs(t, err, s3lock.ErrAlreadyUnlocked)
}
