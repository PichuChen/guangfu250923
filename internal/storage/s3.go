package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"guangfu250923/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Uploader struct {
	client   *s3.Client
	bucket   string
	baseURL  string
	maxBytes int64
}

func NewS3Uploader(ctx context.Context, cfg config.Config) (*S3Uploader, error) {
	if cfg.S3Bucket == "" {
		return nil, errors.New("S3 bucket not configured")
	}

	// Build AWS config
	var loadOpts []func(*awscfg.LoadOptions) error
	if cfg.S3Region != "" {
		loadOpts = append(loadOpts, awscfg.WithRegion(cfg.S3Region))
	}
	if cfg.S3AccessKey != "" && cfg.S3SecretKey != "" {
		creds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, ""))
		loadOpts = append(loadOpts, awscfg.WithCredentialsProvider(creds))
	}
	if cfg.S3Endpoint != "" {
		// Will override via client options below
	}
	acfg, err := awscfg.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, err
	}

	s3opts := func(o *s3.Options) {
		if cfg.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		}
		if cfg.S3UsePathStyle {
			o.UsePathStyle = true
		}
	}
	client := s3.NewFromConfig(acfg, s3opts)

	maxBytes := int64(cfg.MaxUploadMB) * 1024 * 1024
	return &S3Uploader{client: client, bucket: cfg.S3Bucket, baseURL: cfg.S3BaseURL, maxBytes: maxBytes}, nil
}

// Upload streams the file to S3 and returns public URL (or empty if baseURL unset) and the object key.
func (u *S3Uploader) Upload(ctx context.Context, key string, r io.Reader, contentType string) (url string, objectKey string, err error) {
	if u == nil || u.client == nil {
		return "", "", errors.New("uploader not initialized")
	}
	if key == "" {
		return "", "", errors.New("key required")
	}

	// Optional size limiter: wrap reader
	lr := io.LimitedReader{R: r, N: u.maxBytes + 1}

	up := manager.NewUploader(u.client)
	out, err := up.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(key),
		Body:        &lr,
		ContentType: aws.String(contentType),
		ACL:         s3types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return "", "", err
	}

	objKey := key
	if u.baseURL != "" {
		base := strings.TrimRight(u.baseURL, "/")
		url = base + "/" + strings.TrimLeft(objKey, "/")
	} else {
		// fallback to Location if provided
		url = out.Location
	}
	return url, objKey, nil
}

// MaxBytes returns the maximum upload size in bytes configured for this uploader.
func (u *S3Uploader) MaxBytes() int64 { return u.maxBytes }

// PresignGet generates a time-limited URL for downloading the object.
func (u *S3Uploader) PresignGet(ctx context.Context, key string, expires time.Duration) (string, error) {
	if u == nil || u.client == nil {
		return "", errors.New("uploader not initialized")
	}
	if key == "" {
		return "", errors.New("key required")
	}
	presigner := s3.NewPresignClient(u.client, func(o *s3.PresignOptions) { o.Expires = expires })
	out, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &u.bucket,
		Key:    &key,
	})
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

// GetObject fetches an object body for server-side consumption. Caller must Close the body.
func (u *S3Uploader) GetObject(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	if u == nil || u.client == nil {
		return nil, "", 0, errors.New("uploader not initialized")
	}
	if key == "" {
		return nil, "", 0, errors.New("key required")
	}
	out, err := u.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &u.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, "", 0, err
	}
	ctype := ""
	if out.ContentType != nil {
		ctype = *out.ContentType
	}
	var clen int64 = -1
	if out.ContentLength != nil {
		clen = *out.ContentLength
	}
	return out.Body, ctype, clen, nil
}
