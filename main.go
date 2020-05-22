// Copyright 2017 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"log"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsouza/s3-upload-proxy/internal/cachecontrol"
	"github.com/fsouza/s3-upload-proxy/internal/uploader"
	"github.com/fsouza/s3-upload-proxy/internal/uploader/mediastore"
	"github.com/fsouza/s3-upload-proxy/internal/uploader/s3"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

// Config is the configuration of the s3-uploader.
type Config struct {
	BucketName        string             `envconfig:"BUCKET_NAME" required:"true"`
	S3Region          string             `envconfig:"S3_REGION" required:"true"`
	S3IsImplicitAuth  bool               `envconfig:"S3_IS_IMPLICIT_AUTH" default:"true"`
	S3Endpoint        string             `envconfig:"S3_ENDPOINT" default:""`
	S3AccessKeyID     string             `envconfig:"S3_ACCESS_KEY_ID" default:""`
	S3SecretAccessKey string             `envconfig:"S3_SECRET_ACCESS_KEY" default:""`
	UploadDriver      string             `envconfig:"UPLOAD_DRIVER" default:"s3"`
	HealthcheckPath   string             `envconfig:"HEALTHCHECK_PATH" default:"/healthcheck"`
	HTTPPort          int                `envconfig:"HTTP_PORT" default:"80"`
	LogLevel          string             `envconfig:"LOG_LEVEL" default:"debug"`
	CacheControl      cachecontrol.Rules `envconfig:"CACHE_CONTROL_RULES"`
}

func loadConfig() (Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return cfg, err
	}
	if cfg.UploadDriver != "s3" && cfg.UploadDriver != "mediastore" {
		return cfg, errors.New(`invalid UPLOAD_DRIVER, valid options are "s3" and "mediastore"`)
	}
	return cfg, nil
}

func (c *Config) uploader() (uploader.Uploader, error) {
	if c.UploadDriver == "s3" {
		if c.S3IsImplicitAuth {
			return s3.New(s3.S3Options{
				IsLocal: false,
				Region:  c.S3Region,
			})
		} else {
			return s3.New(s3.S3Options{
				IsLocal:     true,
				Region:      c.S3Region,
				Endpoint:    &c.S3Endpoint,
				Credentials: credentials.NewStaticCredentials(c.S3AccessKeyID, c.S3SecretAccessKey, ""),
			})
		}
	}
	if c.UploadDriver == "mediastore" {
		return mediastore.New()
	}
	return nil, fmt.Errorf("invalid upload driver %q", c.UploadDriver)
}

func (c *Config) logger() *logrus.Logger {
	level, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		level = logrus.DebugLevel
	}
	logger := logrus.New()
	logger.Level = level
	return logger
}

func healthcheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	logger := cfg.logger()

	uper, err := cfg.uploader()
	if err != nil {
		logger.WithError(err).Fatal("failed to create uploader")
	}

	http.HandleFunc(cfg.HealthcheckPath, healthcheck)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer r.Body.Close()
		if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		key := strings.TrimLeft(r.URL.Path, "/")
		contentType := mime.TypeByExtension(filepath.Ext(key))
		logFields := logrus.Fields{"bucket": cfg.BucketName, "objectKey": key, "contentType": contentType}
		options := uploader.Options{
			Bucket:       cfg.BucketName,
			Path:         key,
			Body:         r.Body,
			ContentType:  stringPtr(contentType),
			Context:      r.Context(),
			CacheControl: cfg.CacheControl.HeaderValue(key),
		}
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			err = uper.Upload(options)
			if err != nil {
				logger.WithFields(logFields).WithError(err).Error("failed to upload file")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			logger.WithFields(logFields).Debugf("finished upload in %s", time.Since(start))
		case http.MethodDelete:
			err = uper.Delete(options)
			if err != nil {
				logger.WithFields(logFields).WithError(err).Error("failed to delete file")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			logger.WithFields(logFields).Debugf("deleted in %s", time.Since(start))
		}
		fmt.Fprintln(w, "OK")
	})

	listenAddr := fmt.Sprintf(":%d", cfg.HTTPPort)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.WithError(err).Fatal("failed to start listener")
	}
	defer listener.Close()
	logger.Infof("listening on %s", listener.Addr())
	http.Serve(listener, nil)
}

// stringPtr makes empty strings a nil pointer.
func stringPtr(input string) *string {
	if input == "" {
		return nil
	}
	return &input
}
