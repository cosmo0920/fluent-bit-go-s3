package main

import "github.com/fluent/fluent-bit-go/output"
import "github.com/json-iterator/go"
import "github.com/aws/aws-sdk-go/aws"
import "github.com/aws/aws-sdk-go/aws/session"
import "github.com/aws/aws-sdk-go/service/s3/s3manager"
import "github.com/prometheus/common/version"

import (
	"C"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var plugin GoOutputPlugin = &fluentPlugin{}

type s3 struct {
	bucket         string
	prefix         string
	uploader       *s3manager.Uploader
	compressFormat format
}

var s3operator s3

type GoOutputPlugin interface {
	PluginConfigKey(ctx unsafe.Pointer, key string) string
	Unregister(ctx unsafe.Pointer)
	GetRecord(dec *output.FLBDecoder) (ret int, ts interface{}, rec map[interface{}]interface{})
	NewDecoder(data unsafe.Pointer, length int) *output.FLBDecoder
	Put(objectKey string, timestamp time.Time, line string) error
	Exit(code int)
}

type fluentPlugin struct{}

func (p *fluentPlugin) PluginConfigKey(ctx unsafe.Pointer, key string) string {
	return output.FLBPluginConfigKey(ctx, key)
}

func (p *fluentPlugin) Unregister(ctx unsafe.Pointer) {
	output.FLBPluginUnregister(ctx)
}

func (p *fluentPlugin) GetRecord(dec *output.FLBDecoder) (int, interface{}, map[interface{}]interface{}) {
	return output.GetRecord(dec)
}

func (p *fluentPlugin) NewDecoder(data unsafe.Pointer, length int) *output.FLBDecoder {
	return output.NewDecoder(data, int(length))
}

func (p *fluentPlugin) Exit(code int) {
	os.Exit(code)
}

func (p *fluentPlugin) Put(objectKey string, timestamp time.Time, line string) error {
	switch s3operator.compressFormat {
	case plainTextFormat:
		_, err := s3operator.uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(s3operator.bucket),
			Key:    aws.String(objectKey),
			Body:   strings.NewReader(line),
		})
		return err
	case gzipFormat:
		compressed, err := makeGzip([]byte(line))
		_, err = s3operator.uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(s3operator.bucket),
			Key:    aws.String(objectKey),
			Body:   bytes.NewReader(compressed),
		})
		return err
	}

	return nil
}

// based on https://text.baldanders.info/golang/gzip-operation/
func makeGzip(body []byte) ([]byte, error) {
	var b bytes.Buffer
	err := func() error {
		gw := gzip.NewWriter(&b)
		gw.Name = "fluent-bit-go-s3"
		gw.ModTime = time.Now()

		defer gw.Close()

		if _, err := gw.Write(body); err != nil {
			return err
		}
		return nil
	}()
	return b.Bytes(), err
}

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "s3", "S3 Output plugin written in GO!")
}

//export FLBPluginInit
// (fluentbit will call this)
// ctx (context) pointer to fluentbit context (state/ c code)
func FLBPluginInit(ctx unsafe.Pointer) int {
	// Example to retrieve an optional configuration parameter
	credential := plugin.PluginConfigKey(ctx, "Credential")
	accessKeyID := plugin.PluginConfigKey(ctx, "AccessKeyID")
	secretAccessKey := plugin.PluginConfigKey(ctx, "SecretAccessKey")
	bucket := plugin.PluginConfigKey(ctx, "Bucket")
	s3prefix := plugin.PluginConfigKey(ctx, "S3Prefix")
	region := plugin.PluginConfigKey(ctx, "Region")
	compress := plugin.PluginConfigKey(ctx, "Compress")

	config, err := getS3Config(accessKeyID, secretAccessKey, credential, s3prefix, bucket, region, compress)
	if err != nil {
		plugin.Unregister(ctx)
		plugin.Exit(1)
		return output.FLB_ERROR
	}
	fmt.Printf("[flb-go] Starting fluent-bit-go-s3: %s\n", version.Info())
	fmt.Printf("[flb-go] plugin credential parameter = '%s'\n", credential)
	fmt.Printf("[flb-go] plugin accessKeyID parameter = '%s'\n", accessKeyID)
	fmt.Printf("[flb-go] plugin secretAccessKey parameter = '%s'\n", secretAccessKey)
	fmt.Printf("[flb-go] plugin bucket parameter = '%s'\n", bucket)
	fmt.Printf("[flb-go] plugin s3prefix parameter = '%s'\n", s3prefix)
	fmt.Printf("[flb-go] plugin region parameter = '%s'\n", region)
	fmt.Printf("[flb-go] plugin compress parameter = '%s'\n", compress)

	sess := session.New(&aws.Config{
		Credentials: config.credentials,
		Region:      config.region,
	})

	uploader := s3manager.NewUploader(sess, func(u *s3manager.Uploader) {
		u.PartSize = 5 * 1024 * 1024
		u.LeavePartsOnError = true
	})

	s3operator = s3{
		bucket:         *config.bucket,
		prefix:         *config.s3prefix,
		uploader:       uploader,
		compressFormat: config.compress,
	}

	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	var ret int
	var record map[interface{}]interface{}

	dec := plugin.NewDecoder(data, int(length))
	var lines string

	for {
		ret, _, record = plugin.GetRecord(dec)
		if ret != 0 {
			break
		}

		line, err := createJSON(record)
		if err != nil {
			fmt.Printf("error creating message for S3: %v\n", err)
			continue
		}
		lines += line + "\n"
	}

	objectKey := GenerateObjectKey(s3operator.bucket, time.Now())
	err := plugin.Put(objectKey, time.Now(), lines)
	if err != nil {
		fmt.Printf("error sending message for S3: %v\n", err)
		return output.FLB_RETRY
	}

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

// format is S3_PREFIX/S3_TRAILING_PREFIX/date/hour/timestamp_uuid.log
func GenerateObjectKey(S3Prefix string, t time.Time) string {
	var fileext string
	switch s3operator.compressFormat {
	case plainTextFormat:
		fileext = ".log"
	case gzipFormat:
		fileext = ".log.gz"
	}
	timestamp := t.Format("20060102150405")
	date := t.Format("20060102")
	hour := strconv.Itoa(t.Hour())

	fileName := strings.Join([]string{timestamp, fileext}, "")

	objectKey := filepath.Join(s3operator.prefix, date, hour, fileName)
	return objectKey
}

func encodeJSON(record map[interface{}]interface{}) map[string]interface{} {
	m := make(map[string]interface{})

	for k, v := range record {
		switch t := v.(type) {
		case []byte:
			// prevent encoding to base64
			m[k.(string)] = string(t)
		case map[interface{}]interface{}:
			if nextValue, ok := record[k].(map[interface{}]interface{}); ok {
				m[k.(string)] = encodeJSON(nextValue)
			}
		default:
			m[k.(string)] = v
		}
	}

	return m
}

func createJSON(record map[interface{}]interface{}) (string, error) {
	m := encodeJSON(record)

	js, err := jsoniter.Marshal(m)
	if err != nil {
		return "{}", err
	}

	return string(js), nil
}

//export FLBPluginExit
func FLBPluginExit() int {
	return output.FLB_OK
}

func main() {
}
