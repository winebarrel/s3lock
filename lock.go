package s3lock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
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

func (obj *Object) Lock(ctx context.Context) (*lock, error) {
	id := uuid.NewString()

	input := &s3.PutObjectInput{
		Body:        strings.NewReader(id),
		Bucket:      aws.String(obj.bucket),
		Key:         aws.String(obj.key),
		IfNoneMatch: aws.String("*"),
	}

	output, err := obj.s3.PutObject(ctx, input)

	if err != nil {
		return nil, err
	}

	l := &lock{
		s3:     obj.s3,
		bucket: obj.bucket,
		key:    obj.key,
		id:     id,
		etag:   aws.ToString(output.ETag),
	}

	return l, err
}

type lock struct {
	s3     *s3.Client
	bucket string
	key    string
	id     string
	etag   string
}

func (l *lock) validate(ctx context.Context) error {
	input := &s3.GetObjectInput{
		Bucket:  aws.String(l.bucket),
		Key:     aws.String(l.key),
		IfMatch: aws.String(l.etag),
	}

	output, err := l.s3.GetObject(ctx, input)

	if err != nil {
		return err
	}

	defer output.Body.Close()

	b, err := io.ReadAll(output.Body)

	if err != nil {
		return err
	}

	body := string(b)

	if body != l.id {
		return fmt.Errorf("lock id does not match, expected '%s' but got '%s'", l.id, body)
	}

	return nil
}

func (l *lock) Unlock(ctx context.Context) error {
	err := l.validate(ctx)

	if err != nil {
		return err
	}

	input := &s3.DeleteObjectInput{
		Bucket:  aws.String(l.bucket),
		Key:     aws.String(l.key),
		IfMatch: aws.String(l.etag),
	}

	_, err = l.s3.DeleteObject(ctx, input)

	return err
}

type lockJSON struct {
	Bucket string
	Key    string
	Id     string
	ETag   string
}

func (l *lock) MarshalJSON() ([]byte, error) {
	j := &lockJSON{
		Bucket: l.bucket,
		Key:    l.key,
		Id:     l.id,
		ETag:   l.etag,
	}

	return json.Marshal(j)
}

func JSONToLock(s3Client *s3.Client, data []byte) (*lock, error) {
	j := lockJSON{}
	err := json.Unmarshal(data, &j)

	if err != nil {
		return nil, err
	}

	l := &lock{
		s3:     s3Client,
		bucket: j.Bucket,
		key:    j.Key,
		id:     j.Id,
		etag:   j.ETag,
	}

	return l, nil
}
