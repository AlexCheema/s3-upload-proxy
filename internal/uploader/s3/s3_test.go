package s3_test

import (
	"bytes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/fsouza/s3-upload-proxy/internal/uploader"
	"github.com/fsouza/s3-upload-proxy/internal/uploader/s3"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestS3Upload(t *testing.T) {
	isImplicitAuth := true

	// explicit auth
	endpoint := "http://localhost:8000"
	accessKeyID := "S3RVER"
	secretAccessKey := "S3RVER"

	var u uploader.Uploader
	var err error
	var bucket string

	if isImplicitAuth {
		bucket = "best-local-s3-test"
		region := "us-east-1"

		u, err = s3.New(s3.S3Options{
			IsLocal: false,
			Region:  region,
		})
		assert.NoError(t, err)
	} else {
		bucket = "local-bucket"
		region := "us-east-1"

		u, err = s3.New(s3.S3Options{
			IsLocal:     true,
			Region:      region,
			Endpoint:    aws.String(endpoint),
			Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
		})
		assert.NoError(t, err)

	}

	err = u.Upload(uploader.Options{
		Context:      nil,
		Bucket:       bucket,
		Path:         "test2.txt",
		Body:         bytes.NewReader([]byte("hellooooo")),
		ContentType:  aws.String("text/plain"),
		CacheControl: nil,
	})
	assert.NoError(t, err)
}
