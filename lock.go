package s3lock

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
)

var (
	ErrLockAlreadyHeld = errors.New("lock already held")
	ErrAlreadyUnlocked = errors.New("already unlocked")
	ErrLockMismatch    = errors.New("lock mismatch")
)

type Object struct {
	s3     *s3.Client
	bucket string
	key    string
}

func New(s3Client *s3.Client, bucket string, key string) *Object {
	obj := &Object{
		s3:     s3Client,
		bucket: bucket,
		key:    key,
	}

	return obj
}

func (obj *Object) Lock(ctx context.Context) (*Lock, error) {
	id := uuid.NewString()

	input := &s3.PutObjectInput{
		Body:        strings.NewReader(id),
		Bucket:      aws.String(obj.bucket),
		Key:         aws.String(obj.key),
		IfNoneMatch: aws.String("*"),
	}

	output, err := obj.s3.PutObject(ctx, input)

	if err != nil {
		var (
			opeErr  *smithy.OperationError
			respErr *awshttp.ResponseError
		)

		if errors.As(err, &opeErr) && errors.As(opeErr, &respErr) &&
			respErr.Response.StatusCode == http.StatusPreconditionFailed {
			return nil, ErrLockAlreadyHeld
		}

		return nil, err
	}

	l := &Lock{
		s3:     obj.s3,
		bucket: obj.bucket,
		key:    obj.key,
		id:     id,
		etag:   aws.ToString(output.ETag),
	}

	return l, nil
}

type Lock struct {
	mu       sync.Mutex
	unlocked bool
	s3       *s3.Client
	bucket   string
	key      string
	id       string
	etag     string
}

func (l *Lock) String() string {
	return fmt.Sprintf("s3://%s/%s", l.bucket, l.key)
}

func (l *Lock) validate(ctx context.Context) error {
	if l.unlocked {
		return ErrAlreadyUnlocked
	}

	input := &s3.GetObjectInput{
		Bucket:  aws.String(l.bucket),
		Key:     aws.String(l.key),
		IfMatch: aws.String(l.etag),
	}

	output, err := l.s3.GetObject(ctx, input)

	if err != nil {
		var (
			opeErr  *smithy.OperationError
			respErr *awshttp.ResponseError
		)

		if errors.As(err, &opeErr) && errors.As(opeErr, &respErr) {
			switch respErr.Response.StatusCode {
			case http.StatusNotFound:
				return ErrAlreadyUnlocked
			case http.StatusPreconditionFailed:
				return ErrLockMismatch
			}
		}

		return err
	}

	defer output.Body.Close() //nolint:errcheck

	b, err := io.ReadAll(output.Body)

	if err != nil {
		return err
	}

	if string(b) != l.id {
		return ErrLockMismatch
	}

	return nil
}

func (l *Lock) Unlock() error {
	return l.UnlockContext(context.Background())
}

func (l *Lock) UnlockContext(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.validate(ctx); err != nil {
		return err
	}

	input := &s3.DeleteObjectInput{
		Bucket:  aws.String(l.bucket),
		Key:     aws.String(l.key),
		IfMatch: aws.String(l.etag),
	}

	_, err := l.s3.DeleteObject(ctx, input)

	if err == nil {
		l.unlocked = true
	}

	return err
}

type lockJSON struct {
	Bucket string
	Key    string
	Id     string
	ETag   string
}

func (l *Lock) MarshalJSON() ([]byte, error) {
	j := &lockJSON{
		Bucket: l.bucket,
		Key:    l.key,
		Id:     l.id,
		ETag:   l.etag,
	}

	return json.Marshal(j)
}

func NewLockFromJSON(s3Client *s3.Client, data []byte) (*Lock, error) {
	j := lockJSON{}
	err := json.Unmarshal(data, &j)

	if err != nil {
		return nil, err
	}

	l := &Lock{
		s3:     s3Client,
		bucket: j.Bucket,
		key:    j.Key,
		id:     j.Id,
		etag:   j.ETag,
	}

	return l, nil
}

var LockWaitInterval = 1 * time.Second

func (obj *Object) LockWait(ctx context.Context) (*Lock, error) {
	// first time
	lock, err := obj.Lock(ctx)

	if err == nil {
		return lock, nil
	}

	if !errors.Is(err, ErrLockAlreadyHeld) {
		return nil, err
	}

	// after the second time
	ticker := time.NewTicker(LockWaitInterval)
	defer ticker.Stop()
	lastErr := err

L:
	for {
		select {
		case <-ctx.Done():
			break L
		case <-ticker.C:
			lock, err := obj.Lock(ctx)

			if err == nil {
				return lock, nil
			}

			if !errors.Is(err, ErrLockAlreadyHeld) {
				return nil, err
			}

			lastErr = err
		}
	}

	return nil, lastErr
}
