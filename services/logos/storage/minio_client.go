package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sagarjhaa/localfinance/services/logos/config"
)

type MinIOClient struct {
	client *minio.Client
	bucket string
}

func NewMinIOClient(cfg config.StorageConfig) (*MinIOClient, error) {
	// Initialize MinIO client
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	minioClient := &MinIOClient{
		client: client,
		bucket: cfg.Bucket,
	}

	// Ensure bucket exists
	if err := minioClient.ensureBucketExists(); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return minioClient, nil
}

func (mc *MinIOClient) ensureBucketExists() error {
	ctx := context.Background()
	
	// Check if bucket exists
	exists, err := mc.client.BucketExists(ctx, mc.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	// Create bucket if it doesn't exist
	if !exists {
		err = mc.client.MakeBucket(ctx, mc.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

func (mc *MinIOClient) UploadFile(objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	ctx := context.Background()
	
	// Upload file to MinIO
	_, err := mc.client.PutObject(ctx, mc.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Return the object URL
	url := fmt.Sprintf("http://%s/%s/%s", mc.client.EndpointURL().Host, mc.bucket, objectName)
	
	return url, nil
}

func (mc *MinIOClient) DownloadFile(objectName string) (io.Reader, error) {
	ctx := context.Background()
	
	// Download file from MinIO
	object, err := mc.client.GetObject(ctx, mc.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	return object, nil
}

func (mc *MinIOClient) DeleteFile(objectName string) error {
	ctx := context.Background()
	
	// Delete file from MinIO
	err := mc.client.RemoveObject(ctx, mc.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (mc *MinIOClient) ListFiles(prefix string) ([]string, error) {
	ctx := context.Background()
	
	var files []string
	
	// List objects with prefix
	objectCh := mc.client.ListObjects(ctx, mc.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}
		files = append(files, object.Key)
	}

	return files, nil
}

func (mc *MinIOClient) GetFileInfo(objectName string) (*minio.ObjectInfo, error) {
	ctx := context.Background()
	
	// Get object information
	objInfo, err := mc.client.StatObject(ctx, mc.bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &objInfo, nil
}