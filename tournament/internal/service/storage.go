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
	r2config "github.com/braccet/tournament/internal/config"
)

// PresignedUploadResponse contains the presigned URL and final logo URL
type PresignedUploadResponse struct {
	UploadURL string `json:"upload_url"`
	LogoURL   string `json:"logo_url"`
	ExpiresAt string `json:"expires_at"`
}

// StorageService handles file upload operations to R2
type StorageService interface {
	// GenerateEventLogoUploadURL creates a presigned URL for uploading an event logo
	GenerateEventLogoUploadURL(ctx context.Context, eventID uint64, contentType string) (*PresignedUploadResponse, error)

	// GenerateTournamentLogoUploadURL creates a presigned URL for uploading a tournament logo
	GenerateTournamentLogoUploadURL(ctx context.Context, tournamentID uint64, contentType string) (*PresignedUploadResponse, error)

	// ValidateContentType checks if the content type is allowed for logo uploads
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

// GenerateEventLogoUploadURL creates a presigned URL for uploading an event logo
func (s *storageService) GenerateEventLogoUploadURL(ctx context.Context, eventID uint64, contentType string) (*PresignedUploadResponse, error) {
	if !s.ValidateContentType(contentType) {
		return nil, fmt.Errorf("invalid content type: %s", contentType)
	}

	// Determine file extension from content type
	ext := contentTypeToExt(contentType)

	// Build the object key: [prefix/]events/{event_id}/logo.{ext}
	var key string
	if s.prefix != "" {
		key = fmt.Sprintf("%s/events/%d/logo%s", s.prefix, eventID, ext)
	} else {
		key = fmt.Sprintf("events/%d/logo%s", eventID, ext)
	}

	return s.generatePresignedURL(ctx, key, contentType)
}

// GenerateTournamentLogoUploadURL creates a presigned URL for uploading a tournament logo
func (s *storageService) GenerateTournamentLogoUploadURL(ctx context.Context, tournamentID uint64, contentType string) (*PresignedUploadResponse, error) {
	if !s.ValidateContentType(contentType) {
		return nil, fmt.Errorf("invalid content type: %s", contentType)
	}

	// Determine file extension from content type
	ext := contentTypeToExt(contentType)

	// Build the object key: [prefix/]tournaments/{tournament_id}/logo.{ext}
	var key string
	if s.prefix != "" {
		key = fmt.Sprintf("%s/tournaments/%d/logo%s", s.prefix, tournamentID, ext)
	} else {
		key = fmt.Sprintf("tournaments/%d/logo%s", tournamentID, ext)
	}

	return s.generatePresignedURL(ctx, key, contentType)
}

func (s *storageService) generatePresignedURL(ctx context.Context, key, contentType string) (*PresignedUploadResponse, error) {
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

	// Build the public URL for accessing the logo after upload
	logoURL := s.buildPublicURL(key)

	return &PresignedUploadResponse{
		UploadURL: presignResult.URL,
		LogoURL:   logoURL,
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

func (s *noopStorageService) GenerateEventLogoUploadURL(ctx context.Context, eventID uint64, contentType string) (*PresignedUploadResponse, error) {
	return nil, fmt.Errorf("storage service not configured: missing R2 credentials")
}

func (s *noopStorageService) GenerateTournamentLogoUploadURL(ctx context.Context, tournamentID uint64, contentType string) (*PresignedUploadResponse, error) {
	return nil, fmt.Errorf("storage service not configured: missing R2 credentials")
}

func (s *noopStorageService) ValidateContentType(contentType string) bool {
	return false
}
