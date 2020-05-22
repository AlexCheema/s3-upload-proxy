// Copyright 2018 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package s3

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/fsouza/s3-upload-proxy/internal/uploader"
)

type S3Options struct {
	Region      string
	IsLocal     bool
	Endpoint    *string
	Credentials *credentials.Credentials
}

// New returns an uploader that sends objects to S3.
func New(opts S3Options) (uploader.Uploader, error) {
	u := s3Uploader{}
	var cfg *aws.Config = nil
	if opts.IsLocal {
		cfg = &aws.Config{
			Credentials:      opts.Credentials,
			Endpoint:         aws.String(*opts.Endpoint),
			Region:           aws.String(opts.Region),
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(true),
		}
	} else {
		cfg = &aws.Config{
			Region: aws.String(opts.Region),
		}
	}

	sess, _ := session.NewSession(cfg)
	s3Client := s3.New(sess)
	u.client = s3Client

	u.uploader = s3manager.NewUploaderWithClient(s3Client)
	return &u, nil
}

type s3Uploader struct {
	client   *s3.S3
	uploader *s3manager.Uploader
}

func (u *s3Uploader) Upload(options uploader.Options) error {
	input := s3manager.UploadInput{
		Bucket:       aws.String(options.Bucket),
		Key:          aws.String(options.Path),
		Body:         options.Body,
		ContentType:  options.ContentType,
		CacheControl: options.CacheControl,
	}
	_, err := u.uploader.Upload(&input)
	return err
}

func (u *s3Uploader) Delete(options uploader.Options) error {
	req, _ := u.client.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket:                    aws.String(options.Bucket),
		BypassGovernanceRetention: nil,
		Key:                       aws.String(options.Path),
		MFA:                       nil,
		RequestPayer:              nil,
		VersionId:                 nil,
	})
	req.SetContext(options.Context)
	err := req.Send()
	return err
}
