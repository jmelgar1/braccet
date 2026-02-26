package config

import "fmt"

// R2Config holds Cloudflare R2 configuration (S3-compatible)
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Prefix          string // Optional folder prefix within the bucket
	PublicURL       string // Public URL for serving files
}

func LoadR2Config() R2Config {
	return R2Config{
		AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		Bucket:          getEnv("R2_BUCKET", "braccet-uploads"),
		Prefix:          getEnv("R2_PREFIX", ""),
		PublicURL:       getEnv("R2_PUBLIC_URL", ""),
	}
}

// Endpoint returns the R2 S3-compatible endpoint URL
func (c R2Config) Endpoint() string {
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.AccountID)
}

// IsConfigured returns true if R2 credentials are set
func (c R2Config) IsConfigured() bool {
	return c.AccountID != "" && c.AccessKeyID != "" && c.SecretAccessKey != ""
}
