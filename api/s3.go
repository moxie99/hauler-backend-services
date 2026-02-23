package api

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

var s3Client *s3.Client
var s3Bucket string
var s3Region string

// InitS3 initializes the AWS S3 client using environment variables.
// Required env vars: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION, AWS_S3_BUCKET
func InitS3() {
	s3Bucket = os.Getenv("AWS_S3_BUCKET")
	if s3Bucket == "" {
		log.Println("Warning: AWS_S3_BUCKET not set, file uploads will fail")
		return
	}

	s3Region = os.Getenv("AWS_REGION")
	if s3Region == "" {
		s3Region = "us-east-1"
	}

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if accessKey == "" || secretKey == "" {
		log.Println("Warning: AWS credentials not set, file uploads will fail")
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(s3Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		log.Printf("Warning: Failed to load AWS config: %v", err)
		return
	}

	s3Client = s3.NewFromConfig(cfg)
	log.Println("AWS S3 client initialized successfully")
}

// UploadFileToS3 uploads a file to S3 and returns the public URL.
// The folder parameter organizes files (e.g. "selfies", "documents").
func UploadFileToS3(file *multipart.FileHeader, folder string) (string, error) {
	if s3Client == nil {
		return "", fmt.Errorf("S3 client not initialized. Check AWS environment variables")
	}

	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	ext := strings.ToLower(filepath.Ext(file.Filename))
	key := fmt.Sprintf("%s/%s%s", folder, uuid.New().String(), ext)

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s3Bucket),
		Key:         aws.String(key),
		Body:        src,
		ContentType: aws.String(contentType),
		// No ACL - files remain private
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Return the S3 key instead of full URL (we'll generate pre-signed URLs when needed)
	return key, nil
}

// GeneratePresignedURL generates a temporary URL for accessing a private S3 object.
// This is more secure than making files public. URL expires after the specified duration.
func GeneratePresignedURL(s3Key string, expirationMinutes int) (string, error) {
	if s3Client == nil {
		return "", fmt.Errorf("S3 client not initialized")
	}

	if s3Key == "" {
		return "", nil // Return empty string for empty keys
	}

	presignClient := s3.NewPresignClient(s3Client)
	
	request, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(s3Key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expirationMinutes) * time.Minute
	})
	
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

// ConvertProfileURLsToPresigned converts all S3 keys in a DriverProfile to pre-signed URLs
func ConvertProfileURLsToPresigned(profile *DriverProfile, expirationMinutes int) error {
	var err error
	
	// Step 2: Selfie
	if profile.SelfieURL != "" {
		profile.SelfieURL, err = GeneratePresignedURL(profile.SelfieURL, expirationMinutes)
		if err != nil {
			log.Printf("Failed to generate presigned URL for selfie: %v", err)
		}
	}
	
	// Step 3: License and Vehicle Documents
	if profile.LicenseFrontURL != "" {
		profile.LicenseFrontURL, err = GeneratePresignedURL(profile.LicenseFrontURL, expirationMinutes)
		if err != nil {
			log.Printf("Failed to generate presigned URL for license front: %v", err)
		}
	}
	
	if profile.LicenseBackURL != "" {
		profile.LicenseBackURL, err = GeneratePresignedURL(profile.LicenseBackURL, expirationMinutes)
		if err != nil {
			log.Printf("Failed to generate presigned URL for license back: %v", err)
		}
	}
	
	if profile.VehiclePhotoURL != "" {
		profile.VehiclePhotoURL, err = GeneratePresignedURL(profile.VehiclePhotoURL, expirationMinutes)
		if err != nil {
			log.Printf("Failed to generate presigned URL for vehicle photo: %v", err)
		}
	}
	
	if profile.VehicleRegistrationURL != "" {
		profile.VehicleRegistrationURL, err = GeneratePresignedURL(profile.VehicleRegistrationURL, expirationMinutes)
		if err != nil {
			log.Printf("Failed to generate presigned URL for vehicle registration: %v", err)
		}
	}
	
	// Step 5: Insurance and Roadworthiness Documents
	if profile.InsuranceDocumentURL != "" {
		profile.InsuranceDocumentURL, err = GeneratePresignedURL(profile.InsuranceDocumentURL, expirationMinutes)
		if err != nil {
			log.Printf("Failed to generate presigned URL for insurance document: %v", err)
		}
	}
	
	if profile.RoadworthinessDocumentURL != "" {
		profile.RoadworthinessDocumentURL, err = GeneratePresignedURL(profile.RoadworthinessDocumentURL, expirationMinutes)
		if err != nil {
			log.Printf("Failed to generate presigned URL for roadworthiness document: %v", err)
		}
	}
	
	return nil
}

// ValidateImageFile checks that the uploaded file is a valid image and within size limits.
// maxSizeMB is the maximum file size in megabytes.
func ValidateImageFile(file *multipart.FileHeader, maxSizeMB int64) error {
	if file.Size > maxSizeMB*1024*1024 {
		return fmt.Errorf("file size exceeds %dMB limit", maxSizeMB)
	}

	contentType := file.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}

	if !allowedTypes[contentType] {
		return fmt.Errorf("invalid file type. Allowed: JPEG, PNG, WebP")
	}

	return nil
}
