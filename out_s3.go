package main

import "github.com/fluent/fluent-bit-go/output"
import "github.com/json-iterator/go"
import "github.com/google/uuid"
import "github.com/aws/aws-sdk-go/aws"
import "github.com/aws/aws-sdk-go/aws/session"
import "github.com/aws/aws-sdk-go/service/s3/s3manager"

import (
	"C"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var plugin GoOutputPlugin = &fluentPlugin{}

var s3Bucket string
var s3Prefix string
var s3Uploader *s3manager.Uploader

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
	_, err := s3Uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(objectKey),
		Body:   strings.NewReader(line),
	})
	return err
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

	config, err := getS3Config(accessKeyID, secretAccessKey, credential, s3prefix, bucket, region)
	if err != nil {
		plugin.Unregister(ctx)
		plugin.Exit(1)
		return output.FLB_ERROR
	}
	fmt.Printf("[flb-go] plugin credential parameter = '%s'\n", credential)
	fmt.Printf("[flb-go] plugin accessKeyID parameter = '%s'\n", accessKeyID)
	fmt.Printf("[flb-go] plugin secretAccessKey parameter = '%s'\n", secretAccessKey)
	fmt.Printf("[flb-go] plugin bucket parameter = '%s'\n", bucket)
	fmt.Printf("[flb-go] plugin s3prefix parameter = '%s'\n", s3prefix)
	fmt.Printf("[flb-go] plugin region parameter = '%s'\n", region)

	sess := session.New(&aws.Config{
		Credentials: config.credentials,
		Region: config.region,
	})

	uploader := s3manager.NewUploader(sess, func(u *s3manager.Uploader) {
		u.PartSize = 5 * 1024 * 1024
		u.LeavePartsOnError = true
	})

	s3Uploader = uploader
	s3Bucket = *config.bucket
	s3Prefix = *config.s3prefix

	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	var ret int
	var ts interface{}
	var record map[interface{}]interface{}

	dec := plugin.NewDecoder(data, int(length))

	for {
		ret, ts, record = plugin.GetRecord(dec)
		if ret != 0 {
			break
		}

		// Get timestamp
		var timestamp time.Time
		switch t := ts.(type) {
		case output.FLBTime:
			timestamp = ts.(output.FLBTime).Time
		case uint64:
			timestamp = time.Unix(int64(t), 0)
		default:
			fmt.Print("timestamp isn't known format. Use current time.\n")
			timestamp = time.Now()
		}

		line, err := createJSON(record)
		if err != nil {
			fmt.Printf("error creating message for S3: %v\n", err)
			continue
		}

		objectKey := GenerateObjectKey(s3Bucket, timestamp)

		err = plugin.Put(objectKey, timestamp, line)
		if err != nil {
			fmt.Printf("error sending message for S3: %v\n", err)
			return output.FLB_RETRY
		}
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
	timestamp := t.Format("20060102150405")
	date := t.Format("20060102")
	hour := strconv.Itoa(t.Hour())
	logUUID := uuid.Must(uuid.NewRandom()).String()
	fileName := strings.Join([]string{timestamp, "_", logUUID, ".log"}, "")

	objectKey := filepath.Join(S3Prefix, date, hour, fileName)
	return objectKey
}

func createJSON(record map[interface{}]interface{}) (string, error) {
	m := make(map[string]interface{})

	for k, v := range record {
		switch t := v.(type) {
		case []byte:
			// prevent encoding to base64
			m[k.(string)] = string(t)
		default:
			m[k.(string)] = v
		}
	}

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
