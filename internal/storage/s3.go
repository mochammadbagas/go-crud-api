package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string
	endpoint  string
	region    string
}

// S3Config holds configuration options for AWS S3 and Cloudflare R2 storage.
type S3Config struct {
	Bucket          string
	Region          string
	Endpoint        string // e.g. https://<account_id>.r2.cloudflarestorage.com
	AccessKeyID     string
	SecretAccessKey string
	PublicURL       string // e.g. https://pub-xxx.r2.dev or https://cdn.example.com
}

// NewS3Storage creates a storage provider compatible with AWS S3, Cloudflare R2, and MinIO.
func NewS3Storage(ctx context.Context, cfg S3Config) (Storage, error) {
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto" // Default for Cloudflare R2
	}

	awsCfg, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithRegion(region),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS/R2 SDK config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(strings.TrimRight(cfg.Endpoint, "/"))
		}
		o.UsePathStyle = true
	})

	return &s3Storage{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: strings.TrimRight(cfg.PublicURL, "/"),
		endpoint:  strings.TrimRight(cfg.Endpoint, "/"),
		region:    region,
	}, nil
}

// Upload uploads the file stream to S3 / Cloudflare R2 bucket.
func (s *s3Storage) Upload(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
	cleanKey := filepath.Base(filename)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(cleanKey),
		Body:        file,
		ContentType: aws.String(contentType),
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return "", fmt.Errorf("failed to upload file to S3/R2: %w", err)
	}

	// 1. If explicit public URL or custom CDN domain is configured (e.g. Cloudflare R2 custom domain or pub-*.r2.dev)
	if s.publicURL != "" {
		return fmt.Sprintf("%s/%s", s.publicURL, cleanKey), nil
	}

	// 2. If custom endpoint is configured (e.g. MinIO or standard R2 endpoint)
	if s.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucket, cleanKey), nil
	}

	// 3. Fallback to standard AWS S3 format
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, cleanKey), nil
}

// Delete removes an object from the S3 / Cloudflare R2 bucket.
func (s *s3Storage) Delete(ctx context.Context, fileURLOrKey string) error {
	fileURLOrKey = strings.TrimSpace(fileURLOrKey)
	if fileURLOrKey == "" {
		return nil
	}

	// If it's an external HTTP/HTTPS URL that does not belong to this bucket/endpoint/publicURL, skip safely
	if (strings.HasPrefix(fileURLOrKey, "http://") || strings.HasPrefix(fileURLOrKey, "https://")) && !s.isManagedURL(fileURLOrKey) {
		return nil
	}

	cleanKey := ExtractKey(fileURLOrKey)
	if cleanKey == "" {
		return nil
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanKey),
	}

	if _, err := s.client.DeleteObject(ctx, input); err != nil {
		return fmt.Errorf("failed to delete file from S3/R2: %w", err)
	}

	return nil
}

func (s *s3Storage) isManagedURL(fileURLOrKey string) bool {
	if s.publicURL != "" && strings.HasPrefix(fileURLOrKey, s.publicURL) {
		return true
	}
	if s.endpoint != "" && strings.Contains(fileURLOrKey, s.endpoint) {
		return true
	}
	if s.bucket != "" && strings.Contains(fileURLOrKey, s.bucket) {
		return true
	}
	return false
}
