package objectstorage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const maxPhotoSize = int64(50 << 20)

type Store struct {
	client        *s3.Client
	bucket        string
	downloadToken string
}

func New(endpoint, accessKey, secretKey, bucket, downloadToken string) (*Store, error) {
	endpoint = normalizeEndpoint(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("object storage endpoint is empty")
	}

	cfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load object storage client config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})
	return &Store{client: client, bucket: bucket, downloadToken: strings.TrimSpace(downloadToken)}, nil
}

func (s *Store) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err == nil {
		return nil
	}
	_, createErr := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.bucket)})
	if createErr != nil {
		return fmt.Errorf("ensure bucket %q: %w", s.bucket, createErr)
	}
	return nil
}

func (s *Store) UploadURL(ctx context.Context, key, sourceURL, _ string) error {
	if strings.TrimSpace(sourceURL) == "" {
		return fmt.Errorf("photo URL is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return err
	}
	// MAX API requests use the raw bot access token in Authorization (without
	// the Bearer scheme). Most attachment URLs are signed, but protected MAX
	// media URLs may need it. Never forward the bot token to an arbitrary URL
	// received from a user.
	parsedURL, parseErr := url.Parse(sourceURL)
	if parseErr != nil {
		return fmt.Errorf("parse photo URL: %w", parseErr)
	}
	if s.downloadToken != "" && isTrustedMAXHost(parsedURL.Hostname()) {
		req.Header.Set("Authorization", s.downloadToken)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("download photo from MAX: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download photo from MAX: HTTP %s", resp.Status)
	}
	if resp.ContentLength > maxPhotoSize {
		return fmt.Errorf("download photo from MAX: image is larger than 50 MiB")
	}

	photo, err := os.CreateTemp("", "max-photo-*")
	if err != nil {
		return fmt.Errorf("create temporary photo file: %w", err)
	}
	defer func() {
		_ = photo.Close()
		_ = os.Remove(photo.Name())
	}()

	size, err := io.Copy(photo, io.LimitReader(resp.Body, maxPhotoSize+1))
	if err != nil {
		return fmt.Errorf("buffer photo from MAX: %w", err)
	}
	if size > maxPhotoSize {
		return fmt.Errorf("download photo from MAX: image is larger than 50 MiB")
	}
	if _, err := photo.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind temporary photo file: %w", err)
	}

	if err := s.UploadReader(ctx, key, photo, size, resp.Header.Get("Content-Type")); err != nil {
		return fmt.Errorf("upload photo to object storage: %w", err)
	}
	return nil
}

func (s *Store) UploadReader(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   reader,
	}
	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	_, err := s.client.PutObject(ctx, input)
	return err
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" || strings.Contains(endpoint, "://") {
		return endpoint
	}
	return "http://" + endpoint
}

func isTrustedMAXHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	for _, suffix := range []string{"max.ru", "oneme.ru", "okcdn.ru"} {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}
