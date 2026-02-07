package s3lock_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/winebarrel/s3lock"
)

func TestLock(t *testing.T) {
	s3cli := testNewS3Client(t)

	t.Cleanup(func() {
		testDeleteObject(t, s3cli, "s3lock-test", "lock-obj")
	})

	// Lock
	obj := s3lock.New(s3cli, "s3lock-test", "lock-obj")
	lock, err := obj.Lock(t.Context())
	require.NoError(t, err)

	// Confirm that the lock object exists
	body, err := testGetObject(t, s3cli, "s3lock-test", "lock-obj")
	require.NoError(t, err)
	require.Regexp(t, `\w{8}-\w{4}-\w{4}-\w{4}-\w{12}`, body)

	// Unlock
	err = lock.Unlock(t.Context())
	require.NoError(t, err)

	// Confirm that the lock object does not exist
	_, err = testGetObject(t, s3cli, "s3lock-test", "lock-obj")
	require.ErrorContains(t, err, "The specified key does not exist")
}

func TestLockError(t *testing.T) {
	s3cli := testNewS3Client(t)

	t.Cleanup(func() {
		testDeleteObject(t, s3cli, "s3lock-test", "lock-obj")
	})

	obj := s3lock.New(s3cli, "s3lock-test", "lock-obj")

	// Lock
	lock, err := obj.Lock(t.Context())
	require.NoError(t, err)

	// Other clients cannot lock it
	nolock, err := obj.Lock(t.Context())
	require.NoError(t, err)
	require.Nil(t, nolock)

	// Unlock
	err = lock.Unlock(t.Context())
	require.NoError(t, err)

	// Other clients can lock it
	_, err = obj.Lock(t.Context())
	require.NoError(t, err)
}

func TestLockFatal(t *testing.T) {
	s3cli := testNewS3Client(t)

	obj := s3lock.New(s3cli, "xxx-s3lock-test", "lock-obj")

	// Fatal error: bucket does not exist
	lock, err := obj.Lock(t.Context())
	require.ErrorContains(t, err, "The specified bucket does not exist")
	require.Nil(t, lock)
}

func TestMarshalJSON(t *testing.T) {
	s3cli := testNewS3Client(t)

	t.Cleanup(func() {
		testDeleteObject(t, s3cli, "s3lock-test", "lock-obj")
	})

	// Lock
	obj := s3lock.New(s3cli, "s3lock-test", "lock-obj")
	lock, err := obj.Lock(t.Context())
	require.NoError(t, err)

	j, err := json.Marshal(lock)
	require.NoError(t, err)
	require.Regexp(t, `{"Bucket":"s3lock-test","Key":"lock-obj","Id":"\w{8}-\w{4}-\w{4}-\w{4}-\w{12}","ETag":"\\"\w{32}\\""}`, string(j))

	lock, err = s3lock.JSONToLock(s3cli, j)
	require.NoError(t, err)
	j2, err := lock.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, j, j2)

	// Other clients cannot lock it
	nolock, err := obj.Lock(t.Context())
	require.NoError(t, err)
	require.Nil(t, nolock)

	// Unlock
	err = lock.Unlock(t.Context())
	require.NoError(t, err)

	// Confirm that the lock object does not exist
	_, err = testGetObject(t, s3cli, "s3lock-test", "lock-obj")
	require.ErrorContains(t, err, "The specified key does not exist")
}

func TestMD5Collision(t *testing.T) {
	s3cli := testNewS3Client(t)

	t.Cleanup(func() {
		testDeleteObject(t, s3cli, "s3lock-test", "lock-obj")
	})

	id1 := "TEXTCOLLBYfGiJUETHQ4hAcKSMd5zYpgqf1YRDhkmxHkhPWptrkoyz28wnI9V0aHeAuaKnak"
	id2 := "TEXTCOLLBYfGiJUETHQ4hEcKSMd5zYpgqf1YRDhkmxHkhPWptrkoyz28wnI9V0aHeAuaKnak"
	require.NotEqual(t, id1, id2)

	// Manually put the lock object
	etag := *testPutObject(t, s3cli, "s3lock-test", "lock-obj", id1).ETag
	etag = strings.ReplaceAll(etag, `"`, `\"`)

	// Create locks with the same MD5 hash
	lock1, err := s3lock.JSONToLock(s3cli, []byte(`{"Bucket":"s3lock-test","Key":"lock-obj","Id":"`+id1+`","ETag":"`+etag+`"}`))
	require.NoError(t, err)
	lock2, err := s3lock.JSONToLock(s3cli, []byte(`{"Bucket":"s3lock-test","Key":"lock-obj","Id":"`+id2+`","ETag":"`+etag+`"}`))
	require.NoError(t, err)

	// Unlock with a different lock id
	err = lock2.Unlock(t.Context())
	require.ErrorContains(t, err, `lock id does not match, expected 'TEXTCOLLBYfGiJUETHQ4hEcKSMd5zYpgqf1YRDhkmxHkhPWptrkoyz28wnI9V0aHeAuaKnak' but got 'TEXTCOLLBYfGiJUETHQ4hAcKSMd5zYpgqf1YRDhkmxHkhPWptrkoyz28wnI9V0aHeAuaKnak'`)

	// Confirm that the lock object exists
	body, err := testGetObject(t, s3cli, "s3lock-test", "lock-obj")
	require.NoError(t, err)
	require.Regexp(t, id1, body)

	// Unlock with a same lock id
	err = lock1.Unlock(t.Context())
	require.NoError(t, err)

	// Confirm that the lock object does not exist
	_, err = testGetObject(t, s3cli, "s3lock-test", "lock-obj")
	require.ErrorContains(t, err, "The specified key does not exist")
}
