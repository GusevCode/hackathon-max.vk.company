package minio

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Store struct {
	client *minio.Client
	bucket string
}

func New(endpoint, accessKey, secretKey, bucket string) (*Store, error) {
	client, err := minio.New(strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://"), &minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: strings.HasPrefix(endpoint, "https://")})
	if err != nil {
		return nil, fmt.Errorf("create MinIO client: %w", err)
	}
	return &Store{client: client, bucket: bucket}, nil
}

func (s *Store) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
	}
	return nil
}

func (s *Store) UploadURL(ctx context.Context, key, url string) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("photo URL is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download photo: HTTP %s", resp.Status)
	}
	_, err = s.client.PutObject(ctx, s.bucket, key, resp.Body, resp.ContentLength, minio.PutObjectOptions{ContentType: resp.Header.Get("Content-Type")})
	return err
}

func (s *Store) UploadReader(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}
