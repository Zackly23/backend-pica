package config

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	// "github.com/aws/aws-sdk-go-v2/feature/s3/presign"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type AWSS3Bucket struct {
	BucketName string
	Region     string
	S3client   *s3.Client
}

var S3Bucket *AWSS3Bucket

// Setup connection ke S3
func (awsBucket *AWSS3Bucket) SetupBucket() {


	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"",
		)),
	)
	
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	
	S3Bucket = &AWSS3Bucket{
		BucketName: os.Getenv("AWS_BUCKET_NAME"),
		Region:     os.Getenv("AWS_REGION"),
		S3client:   s3.NewFromConfig(cfg),
	}

	awsBucket.Region = os.Getenv("AWS_REGION")
	awsBucket.BucketName = os.Getenv("AWS_BUCKET_NAME")
	awsBucket.S3client = s3.NewFromConfig(cfg)
}


func (awsBucket *AWSS3Bucket) Test() {
	// Test connection with ListBuckets
	result, err := awsBucket.S3client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		log.Fatalf("❌ Gagal koneksi ke S3: %v", err)
	} else {
		log.Println("✅ Koneksi ke S3 berhasil. Bucket tersedia:")
		for _, bucket := range result.Buckets {
			log.Println("🪣", *bucket.Name)
		}
	}

}