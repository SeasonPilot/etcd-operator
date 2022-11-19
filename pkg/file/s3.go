package file

import (
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type s3Uploader struct {
	Endpoint        string
	AccessKeyId     string
	SecretAccessKey string
}

func NewS3Uploader(endpoint string, ak string, sk string) *s3Uploader {
	return &s3Uploader{Endpoint: endpoint, AccessKeyId: ak, SecretAccessKey: sk}
}

func (u *s3Uploader) InitClient() (*minio.Client, error) {
	return minio.New(u.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(u.AccessKeyId, u.SecretAccessKey, ""),
		Secure: true,
	})

}

func (u *s3Uploader) Upload(ctx context.Context, bucketName, objectName, filePath string) (int64, error) {
	c, err := u.InitClient()
	if err != nil {
		panic(err)
	}

	object, err := c.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{})
	if err != nil {
		return 0, err
	}

	return object.Size, nil
}
