package subcmd_test

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
	"github.com/winebarrel/s3lock"
	"github.com/winebarrel/s3lock/cmd/subcmd"
)

func TestLockCmd(t *testing.T) {
	hc := &http.Client{}
	httpmock.ActivateNonDefault(hc)
	t.Cleanup(func() { httpmock.DeactivateNonDefault(hc) })

	cfg, _ := config.LoadDefaultConfig(t.Context(), config.WithHTTPClient(hc))
	s3cli := s3.NewFromConfig(cfg)

	cmd := &subcmd.LockCmd{
		S3URL: &url.URL{Scheme: "s3", Host: "s3lock-test", Path: "/lock-obj"},
	}

	httpmock.RegisterResponder(http.MethodPut, "https://s3lock-test.s3.us-east-1.amazonaws.com/lock-obj?x-id=PutObject", func(req *http.Request) (*http.Response, error) {
		require.Equal(t, `*`, req.Header.Get("If-None-Match"))
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Regexp(t, `\w{8}-\w{4}-\w{4}-\w{4}-\w{12}`, string(body))
		return httpmock.NewStringResponse(http.StatusOK, ""), nil
	})

	var buf bytes.Buffer

	err := cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: &buf,
	})

	require.NoError(t, err)
	require.Regexp(t, `{"Bucket":"s3lock-test","Key":"lock-obj","Id":"\w{8}-\w{4}-\w{4}-\w{4}-\w{12}","ETag":".*"}`, buf.String())
}

func TestLockCmdWithWait(t *testing.T) {
	hc := &http.Client{}
	httpmock.ActivateNonDefault(hc)
	t.Cleanup(func() { httpmock.DeactivateNonDefault(hc) })

	cfg, _ := config.LoadDefaultConfig(t.Context(), config.WithHTTPClient(hc))
	s3cli := s3.NewFromConfig(cfg)

	cmd := &subcmd.LockCmd{
		S3URL: &url.URL{Scheme: "s3", Host: "s3lock-test", Path: "/lock-obj"},
		Wait:  3,
	}

	count := 0

	httpmock.RegisterResponder(http.MethodPut, "https://s3lock-test.s3.us-east-1.amazonaws.com/lock-obj?x-id=PutObject", func(req *http.Request) (*http.Response, error) {
		require.Equal(t, `*`, req.Header.Get("If-None-Match"))
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Regexp(t, `\w{8}-\w{4}-\w{4}-\w{4}-\w{12}`, string(body))

		if count == 0 {
			count++
			return httpmock.NewStringResponse(http.StatusPreconditionFailed, ""), nil
		}

		return httpmock.NewStringResponse(http.StatusOK, ""), nil
	})

	var buf bytes.Buffer

	err := cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: &buf,
	})

	require.NoError(t, err)
	require.Regexp(t, `{"Bucket":"s3lock-test","Key":"lock-obj","Id":"\w{8}-\w{4}-\w{4}-\w{4}-\w{12}","ETag":".*"}`, buf.String())
}

func TestLockCmdLockAlreadyHeld(t *testing.T) {
	hc := &http.Client{}
	httpmock.ActivateNonDefault(hc)
	t.Cleanup(func() { httpmock.DeactivateNonDefault(hc) })

	cfg, _ := config.LoadDefaultConfig(t.Context(), config.WithHTTPClient(hc))
	s3cli := s3.NewFromConfig(cfg)

	cmd := &subcmd.LockCmd{
		S3URL: &url.URL{Scheme: "s3", Host: "s3lock-test", Path: "/lock-obj"},
	}

	httpmock.RegisterResponder(http.MethodPut, "https://s3lock-test.s3.us-east-1.amazonaws.com/lock-obj?x-id=PutObject", func(req *http.Request) (*http.Response, error) {
		require.Equal(t, `*`, req.Header.Get("If-None-Match"))
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Regexp(t, `\w{8}-\w{4}-\w{4}-\w{4}-\w{12}`, string(body))
		return httpmock.NewStringResponse(http.StatusPreconditionFailed, ""), nil
	})

	err := cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: io.Discard,
	})

	require.ErrorIs(t, err, s3lock.ErrLockAlreadyHeld)
}

func TestLockCmdFatal(t *testing.T) {
	hc := &http.Client{}
	httpmock.ActivateNonDefault(hc)
	t.Cleanup(func() { httpmock.DeactivateNonDefault(hc) })

	cfg, _ := config.LoadDefaultConfig(t.Context(), config.WithHTTPClient(hc),
		config.WithRetryer(func() aws.Retryer { return &aws.NopRetryer{} }))
	s3cli := s3.NewFromConfig(cfg)

	cmd := &subcmd.LockCmd{
		S3URL: &url.URL{Scheme: "s3", Host: "s3lock-test", Path: "/lock-obj"},
	}

	httpmock.RegisterResponder(http.MethodPut, "https://s3lock-test.s3.us-east-1.amazonaws.com/lock-obj?x-id=PutObject", func(req *http.Request) (*http.Response, error) {
		require.Equal(t, `*`, req.Header.Get("If-None-Match"))
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Regexp(t, `\w{8}-\w{4}-\w{4}-\w{4}-\w{12}`, string(body))
		return httpmock.NewStringResponse(http.StatusInternalServerError, ""), nil
	})

	err := cmd.Run(&subcmd.Context{
		S3:     s3cli,
		Output: io.Discard,
	})

	require.ErrorContains(t, err, "StatusCode: 500")
}
