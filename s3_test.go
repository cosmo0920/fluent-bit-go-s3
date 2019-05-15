package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
	// "time"
)

func TestGetS3Config(t *testing.T) {
	_, err := getS3Config("exampleaccessID", "examplesecretkey", "examplecredentials", "exampleprefix", "examplebucket", "exampleregion")
	if err != nil {
		t.Fatalf("failed test %#v", err)
	}

	assert.Equal(t, "examplebucket", s3Bucket, "Specify bucket name")
	assert.Equal(t, "exampleprefix", s3Prefix, "Specify s3prefix name")
}
