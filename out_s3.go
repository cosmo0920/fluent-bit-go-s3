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

type GoOutputPlugin interface {
	PluginConfigKey(ctx unsafe.Pointer, key string) string
	Unregister(ctx unsafe.Pointer)
	GetRecord(dec *output.FLBDecoder) (ret int, ts interface{}, rec map[interface{}]interface{})
	NewDecoder(data unsafe.Pointer, length int) *output.FLBDecoder
	Put(s3operator *s3, objectKey string, timestamp time.Time, line string) error
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

func (p *fluentPlugin) Put(s3operator *s3, objectKey string, timestamp time.Time, line string) error {
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
		if err != nil {
			return err
		}
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

var (
	s3operators []*s3
)

func newS3Output(ctx unsafe.Pointer, operatorID int) (*s3, error) {
	// Example to retrieve an optional configuration parameter
	credential := plugin.PluginConfigKey(ctx, "Credential")
	accessKeyID := plugin.PluginConfigKey(ctx, "AccessKeyID")
	secretAccessKey := plugin.PluginConfigKey(ctx, "SecretAccessKey")
	bucket := plugin.PluginConfigKey(ctx, "Bucket")
	s3prefix := plugin.PluginConfigKey(ctx, "S3Prefix")
	region := plugin.PluginConfigKey(ctx, "Region")
	compress := plugin.PluginConfigKey(ctx, "Compress")
	endpoint := plugin.PluginConfigKey(ctx, "Endpoint")

	config, err := getS3Config(accessKeyID, secretAccessKey, credential, s3prefix, bucket, region, compress, endpoint)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[flb-go %d] Starting fluent-bit-go-s3: %s\n", operatorID, version.Info())
	fmt.Printf("[flb-go %d] plugin credential parameter = '%s'\n", operatorID, credential)
	fmt.Printf("[flb-go %d] plugin accessKeyID parameter = '%s'\n", operatorID, accessKeyID)
	fmt.Printf("[flb-go %d] plugin secretAccessKey parameter = '%s'\n", operatorID, secretAccessKey)
	fmt.Printf("[flb-go %d] plugin bucket parameter = '%s'\n", operatorID, bucket)
	fmt.Printf("[flb-go %d] plugin s3prefix parameter = '%s'\n", operatorID, s3prefix)
	fmt.Printf("[flb-go %d] plugin region parameter = '%s'\n", operatorID, region)
	fmt.Printf("[flb-go %d] plugin compress parameter = '%s'\n", operatorID, compress)
	fmt.Printf("[flb-go %d] plugin endpoint parameter = '%s'\n", operatorID, endpoint)

	cfg := aws.Config{
		Credentials: config.credentials,
		Region:      config.region,
	}
	if config.endpoint != "" {
		cfg.WithEndpoint(config.endpoint).WithS3ForcePathStyle(true)
	}

	sess := session.New(&cfg)

	uploader := s3manager.NewUploader(sess, func(u *s3manager.Uploader) {
		u.PartSize = 5 * 1024 * 1024
		u.LeavePartsOnError = true
	})

	s3operator := &s3{
		bucket:         *config.bucket,
		prefix:         *config.s3prefix,
		uploader:       uploader,
		compressFormat: config.compress,
	}

	return s3operator, nil

}

func addS3Output(ctx unsafe.Pointer) error {
	operatorID := len(s3operators)
	fmt.Printf("[s3operator] id = %q\n", operatorID)
	// Set the context to point to any Go variable
	output.FLBPluginSetContext(ctx, operatorID)
	operator, err := newS3Output(ctx, operatorID)
	if err != nil {
		return err
	}

	s3operators = append(s3operators, operator)
	return nil
}

func getS3Operator(ctx unsafe.Pointer) *s3 {
	operatorID := output.FLBPluginGetContext(ctx).(int)
	return s3operators[operatorID]
}

//export FLBPluginInit
// (fluentbit will call this)
// ctx (context) pointer to fluentbit context (state/ c code)
func FLBPluginInit(ctx unsafe.Pointer) int {
	err := addS3Output(ctx)
	if err != nil {
		plugin.Unregister(ctx)
		plugin.Exit(1)
		return output.FLB_ERROR
	}

	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	var ret int
	var record map[interface{}]interface{}

	s3operator := getS3Operator(ctx)
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

	objectKey := GenerateObjectKey(s3operator, time.Now())
	err := plugin.Put(s3operator, objectKey, time.Now(), lines)
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
func GenerateObjectKey(s3operator *s3, t time.Time) string {
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
