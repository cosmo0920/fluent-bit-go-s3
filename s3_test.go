package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
	// "time"
)

func TestGetS3ConfigStaticCredentials(t *testing.T) {
	conf, err := getS3Config("exampleaccessID", "examplesecretkey", "", "exampleprefix", "examplebucket", "exampleregion", "")
	if err != nil {
		t.Fatalf("failed test %#v", err)
	}

	assert.Equal(t, "examplebucket", *conf.bucket, "Specify bucket name")
	assert.Equal(t, "exampleprefix", *conf.s3prefix, "Specify s3prefix name")
	assert.NotNil(t, conf.credentials, "credentials not to be nil")
	assert.Equal(t, "exampleregion", *conf.region, "Specify s3prefix name")
}

func TestGetS3ConfigSharedCredentials(t *testing.T) {
	s3Creds = &testS3Credential{}
	conf, err := getS3Config("", "", "examplecredentials", "exampleprefix", "examplebucket", "exampleregion", "")
	if err != nil {
		t.Fatalf("failed test %#v", err)
	}

	assert.Equal(t, "examplebucket", *conf.bucket, "Specify bucket name")
	assert.Equal(t, "exampleprefix", *conf.s3prefix, "Specify s3prefix name")
	assert.NotNil(t, conf.credentials, "credentials not to be nil")
	assert.Equal(t, "exampleregion", *conf.region, "Specify s3prefix name")
}
