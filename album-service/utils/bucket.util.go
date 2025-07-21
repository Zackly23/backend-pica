package utils

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/Zackly23/queue-app/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func UploadToS3(file *multipart.FileHeader, key string) (string, error) {
	f, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	uploader := manager.NewUploader(config.S3Bucket.S3client)

	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(config.S3Bucket.BucketName),
		Key:    aws.String(key),
		Body:   f,
		// ACL:    "public-read", // sesuaikan dengan permission bucket kamu
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", config.S3Bucket.BucketName, config.S3Bucket.Region, key)
	return url, nil
}

func DeleteFromS3(fileURL string, bucketName string) error {
	client := config.S3Bucket.S3client
	
	// Ambil key dari URL S3
	// Contoh: https://s3-pixovaulty.s3.ap-southeast-1.amazonaws.com/videos/album_xxx/video.mp4
	prefix := fmt.Sprintf("https://%s.s3.ap-southeast-1.amazonaws.com/", bucketName)
	key := strings.TrimPrefix(fileURL, prefix)

	_, err := client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}

	return nil
}

func GeneratePresignedURL(bucketName, key string) (string, error) {
    // Buat presigner langsung dari s3
	
	client :=  config.S3Bucket.S3client

    presigner := s3.NewPresignClient(client)

    // Presign untuk GetObject
    resp, err := presigner.PresignGetObject(context.TODO(), &s3.GetObjectInput{
        Bucket: aws.String(bucketName),
        Key:    aws.String(key),
    }, s3.WithPresignExpires(15*time.Minute))

    if err != nil {
        return "", fmt.Errorf("failed to generate presigned url: %w", err)
    }

    return resp.URL, nil
}

