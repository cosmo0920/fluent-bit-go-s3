package main

import (
	"github.com/aws/aws-sdk-go/aws"
)
import "github.com/aws/aws-sdk-go/aws/credentials"
import log "github.com/sirupsen/logrus"

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type format int

const (
	plainTextFormat format = iota
	gzipFormat
)

type algorithm int

const (
	noSuffixAlgorithm algorithm = iota
	sha256SuffixAlgorithm
)

type s3Config struct {
	credentials      *credentials.Credentials
	bucket           *string
	s3prefix         *string
	suffixAlgorithm  algorithm
	region           *string
	compress         format
	endpoint         string
	logLevel         log.Level
	timeFormat       string
	location         *time.Location
	autoCreateBucket bool
}

type S3Credential interface {
	GetCredentials(accessID, secretkey, credentials string) (*credentials.Credentials, error)
}

type s3PluginConfig struct{}

var s3Creds S3Credential = &s3PluginConfig{}

func (c *s3PluginConfig) GetCredentials(accessKeyID, secretKey, credential string) (*credentials.Credentials, error) {
	if credential != "" {
		creds := credentials.NewSharedCredentials(credential, "default")
		if _, err := creds.Get(); err != nil {
			return nil, fmt.Errorf("[SharedCredentials] ERROR: %s", err)
		}
		return creds, nil
	}
	if !(accessKeyID == "" && secretKey == "") {
		creds := credentials.NewStaticCredentials(accessKeyID, secretKey, "")
		if _, err := creds.Get(); err != nil {
			return nil, fmt.Errorf("[StaticCredentials] ERROR: %s", err)
		}
		return creds, nil

	}
	return nil, nil
}

func getS3Config(accessID, secretKey, credential, s3prefix, suffixAlgorithm, bucket, region, compress, endpoint, autoCreateBucket, logLevel, timeFormat, timeZone string) (*s3Config, error) {
	conf := &s3Config{}
	creds, err := s3Creds.GetCredentials(accessID, secretKey, credential)
	if err != nil {
		return nil, fmt.Errorf("Failed to create credentials")
	}
	conf.credentials = creds

	if bucket == "" {
		return nil, fmt.Errorf("Cannot specify empty string to bucket name")
	}
	conf.bucket = aws.String(bucket)

	if s3prefix == "" {
		return nil, fmt.Errorf("Cannot specify empty string to s3prefix")
	}
	conf.s3prefix = aws.String(s3prefix)

	switch suffixAlgorithm {
	case "sha256":
		conf.suffixAlgorithm = sha256SuffixAlgorithm
	default:
		conf.suffixAlgorithm = noSuffixAlgorithm
	}

	if region == "" {
		return nil, fmt.Errorf("Cannot specify empty string to region")
	}
	conf.region = aws.String(region)

	switch compress {
	case "gzip":
		conf.compress = gzipFormat
	default:
		conf.compress = plainTextFormat
	}

	if endpoint != "" {
		if strings.HasSuffix(endpoint, "amazonaws.com") {
			return nil, fmt.Errorf("Endpoint is not supported for AWS S3. This parameter is intended for S3 compatible services. Use Region instead.")
		}
		conf.endpoint = endpoint
	}

	isAutoCreateBucket, err := strconv.ParseBool(autoCreateBucket)
	if err != nil {
		conf.autoCreateBucket = false
	} else {
		conf.autoCreateBucket = isAutoCreateBucket
	}

	if logLevel == "" {
		logLevel = "info"
	}
	var level log.Level
	if level, err = log.ParseLevel(logLevel); err != nil {
		return nil, fmt.Errorf("invalid log level: %v", logLevel)
	}
	conf.logLevel = level

	if timeFormat != "" {
		conf.timeFormat = timeFormat
	} else {
		conf.timeFormat = "20060102/15"
	}

	if timeZone != "" {
		loc, err := time.LoadLocation(timeZone)
		if err != nil {
			return nil, fmt.Errorf("invalid timeZone: %v", err)
		} else {
			conf.location = loc
		}
	} else {
		conf.location = time.Local
	}

	return conf, nil
}
