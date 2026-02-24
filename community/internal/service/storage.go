package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	r2config "github.com/braccet/community/internal/config"
)

// PresignedUploadResponse contains the presigned URL and final icon URL
type PresignedUploadResponse struct {
	UploadURL string `json:"upload_url"`
	IconURL   string `json:"icon_url"`
	ExpiresAt string `json:"expires_at"`
}

// StorageService handles file upload operations to R2
type StorageService interface {
	// GeneratePresignedUploadURL creates a presigned URL for uploading a member icon
	GeneratePresignedUploadURL(ctx context.Context, communityID, memberID uint64, contentType string) (*PresignedUploadResponse, error)

	// ValidateContentType checks if the content type is allowed for icon uploads
	ValidateContentType(contentType string) bool
}

type storageService struct {
	presignClient *s3.PresignClient
	bucket        string
	prefix        string
	publicURL     string
}

// NewStorageService creates a new storage service for R2
func NewStorageService(cfg r2config.R2Config) (StorageService, error) {
	if !cfg.IsConfigured() {
		return &noopStorageService{}, nil
	}

	// Create custom resolver for R2 endpoint
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: cfg.Endpoint(),
		}, nil
	})

	// Load AWS config with R2 credentials
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
		config.WithRegion("auto"), // R2 uses "auto" as region
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load R2 config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)
	presignClient := s3.NewPresignClient(client)

	return &storageService{
		presignClient: presignClient,
		bucket:        cfg.Bucket,
		prefix:        cfg.Prefix,
		publicURL:     cfg.PublicURL,
	}, nil
}

// GeneratePresignedUploadURL creates a presigned URL for uploading a member icon
func (s *storageService) GeneratePresignedUploadURL(ctx context.Context, communityID, memberID uint64, contentType string) (*PresignedUploadResponse, error) {
	if !s.ValidateContentType(contentType) {
		return nil, fmt.Errorf("invalid content type: %s", contentType)
	}

	// Determine file extension from content type
	ext := contentTypeToExt(contentType)

	// Build the object key: [prefix/]members/{community_id}/{member_id}/icon.{ext}
	var key string
	if s.prefix != "" {
		key = fmt.Sprintf("%s/members/%d/%d/icon%s", s.prefix, communityID, memberID, ext)
	} else {
		key = fmt.Sprintf("members/%d/%d/icon%s", communityID, memberID, ext)
	}

	// Generate presigned URL with 5 minute expiration
	expiresIn := 5 * time.Minute
	expiresAt := time.Now().Add(expiresIn)

	presignResult, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	// Build the public URL for accessing the icon after upload
	iconURL := s.buildPublicURL(key)

	return &PresignedUploadResponse{
		UploadURL: presignResult.URL,
		IconURL:   iconURL,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

// ValidateContentType checks if the content type is allowed
func (s *storageService) ValidateContentType(contentType string) bool {
	allowed := map[string]bool{
		"image/jpeg":    true,
		"image/png":     true,
		"image/webp":    true,
		"image/svg+xml": true,
	}
	return allowed[contentType]
}

// buildPublicURL constructs the public URL for an object
func (s *storageService) buildPublicURL(key string) string {
	// Remove trailing slash from public URL if present
	publicURL := strings.TrimSuffix(s.publicURL, "/")
	return fmt.Sprintf("%s/%s", publicURL, key)
}

// contentTypeToExt converts content type to file extension
func contentTypeToExt(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	default:
		return filepath.Ext(contentType)
	}
}

// noopStorageService is used when R2 is not configured
type noopStorageService struct{}

func (s *noopStorageService) GeneratePresignedUploadURL(ctx context.Context, communityID, memberID uint64, contentType string) (*PresignedUploadResponse, error) {
	return nil, fmt.Errorf("storage service not configured: missing R2 credentials")
}

func (s *noopStorageService) ValidateContentType(contentType string) bool {
	return false
}
