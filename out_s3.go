package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/fluent/fluent-bit-go/output"
)
import "github.com/json-iterator/go"
import "github.com/aws/aws-sdk-go/aws"
import "github.com/aws/aws-sdk-go/aws/awserr"
import "github.com/aws/aws-sdk-go/aws/session"
import "github.com/aws/aws-sdk-go/service/s3"
import "github.com/aws/aws-sdk-go/service/s3/s3manager"
import log "github.com/sirupsen/logrus"
import "github.com/prometheus/common/version"

import (
	"C"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

var plugin GoOutputPlugin = &fluentPlugin{}
var logger *log.Logger
var context GoPluginContext = &pluginContext{}

func init() {
	logLevel, _ := log.ParseLevel("info")
	logger = newLogger(logLevel)
	logger.SetFormatter(new(fluentBitLogFormat))
}

type s3operator struct {
	bucket          string
	prefix          string
	suffixAlgorithm algorithm
	uploader        *s3manager.Uploader
	compressFormat  format
	logger          *log.Logger
	timeFormat      string
	location        *time.Location
}

type GoOutputPlugin interface {
	PluginConfigKey(ctx unsafe.Pointer, key string) string
	Unregister(ctx unsafe.Pointer)
	GetRecord(dec *output.FLBDecoder) (ret int, ts interface{}, rec map[interface{}]interface{})
	NewDecoder(data unsafe.Pointer, length int) *output.FLBDecoder
	Put(s3operator *s3operator, objectKey string, timestamp time.Time, line string) error
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

func (p *fluentPlugin) Put(s3operator *s3operator, objectKey string, timestamp time.Time, line string) error {
	switch s3operator.compressFormat {
	case plainTextFormat:
		s3operator.logger.Tracef("[s3operator] objectKey = %s, rows = %d, byte = %d", objectKey, len(strings.Split(line, "\n")), len(line))
		_, err := s3operator.uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(s3operator.bucket),
			Key:    aws.String(objectKey),
			Body:   strings.NewReader(line),
		})
		return err
	case gzipFormat:
		compressed, err := makeGzip([]byte(line))
		s3operator.logger.Tracef("[s3operator] objectKey = %s, rows = %d, byte = %d", objectKey, len(strings.Split(line, "\n")), len(compressed))
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

type pluginContext struct {}

type GoPluginContext interface {
	PluginGetContext(ctx unsafe.Pointer) interface{}
	PluginSetContext(plugin unsafe.Pointer, ctx interface{})
}

func (p *pluginContext) PluginGetContext(ctx unsafe.Pointer) interface{} {
	return output.FLBPluginGetContext(ctx)
}

func (p *pluginContext) PluginSetContext(plugin unsafe.Pointer, ctx interface{}) {
	output.FLBPluginSetContext(plugin, ctx)
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
	s3operators []*s3operator
)

func ensureBucket(session *session.Session, bucket, region *string) (bool, error) {
	svc := s3.New(session)
	var input *s3.CreateBucketInput
	// us-east-1 is default region. So, it needn't specify region in CreateBucketInput.
	if *region == "us-east-1" {
		input = &s3.CreateBucketInput{
			Bucket: bucket,
		}
	} else {
		input = &s3.CreateBucketInput{
			Bucket: bucket,
			CreateBucketConfiguration: &s3.CreateBucketConfiguration{
				LocationConstraint: region,
			},
		}
	}

	result, err := svc.CreateBucket(input)
	logger.Tracef("CreateBucket request result is: %s, err: %s", result, err)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeBucketAlreadyExists:
				logger.Tracef("Bucket(%s) is already exists.", *bucket)
				return true, nil
			case s3.ErrCodeBucketAlreadyOwnedByYou:
				logger.Tracef("Bucket(%s) is already owned by you.", *bucket)
				return true, nil
			default:
				logger.Tracef("CreateBucket is failed with: %s", aerr.Error())
				return false, aerr
			}
		} else {
			return false, err
		}
	}
	return true, nil
}

func newLogger(logLevel log.Level) *log.Logger {
	logger := log.New()
	logger.Level = logLevel
	logger.SetFormatter(new(fluentBitLogFormat))
	return logger
}

func newS3Output(ctx unsafe.Pointer, operatorID int) (*s3operator, error) {
	// Example to retrieve an optional configuration parameter
	credential := plugin.PluginConfigKey(ctx, "Credential")
	accessKeyID := plugin.PluginConfigKey(ctx, "AccessKeyID")
	secretAccessKey := plugin.PluginConfigKey(ctx, "SecretAccessKey")
	bucket := plugin.PluginConfigKey(ctx, "Bucket")
	s3prefix := plugin.PluginConfigKey(ctx, "S3Prefix")
	suffixAlgorithm := plugin.PluginConfigKey(ctx, "SuffixAlgorithm")
	region := plugin.PluginConfigKey(ctx, "Region")
	compress := plugin.PluginConfigKey(ctx, "Compress")
	endpoint := plugin.PluginConfigKey(ctx, "Endpoint")
	autoCreateBucket := plugin.PluginConfigKey(ctx, "AutoCreateBucket")
	logLevel := plugin.PluginConfigKey(ctx, "LogLevel")
	timeFormat := plugin.PluginConfigKey(ctx, "TimeFormat")
	timeZone := plugin.PluginConfigKey(ctx, "TimeZone")

	config, err := getS3Config(accessKeyID, secretAccessKey, credential, s3prefix, suffixAlgorithm, bucket, region, compress, endpoint, autoCreateBucket, logLevel, timeFormat, timeZone)

	if err != nil {
		return nil, err
	}
	logger := newLogger(config.logLevel)

	logger.Infof("[flb-go %d] Starting fluent-bit-go-s3: %v", operatorID, version.Info())
	logger.Infof("[flb-go %d] plugin credential parameter = '%s'", operatorID, credential)
	logger.Infof("[flb-go %d] plugin accessKeyID parameter = '%s'", operatorID, obfuscateSecret(accessKeyID))
	logger.Infof("[flb-go %d] plugin secretAccessKey parameter = '%s'", operatorID, obfuscateSecret(secretAccessKey))
	logger.Infof("[flb-go %d] plugin bucket parameter = '%s'", operatorID, bucket)
	logger.Infof("[flb-go %d] plugin s3prefix parameter = '%s'", operatorID, s3prefix)
	logger.Infof("[flb-go %d] plugin suffixAlgorithm parameter = '%s'", operatorID, suffixAlgorithm)
	logger.Infof("[flb-go %d] plugin region parameter = '%s'", operatorID, region)
	logger.Infof("[flb-go %d] plugin compress parameter = '%s'", operatorID, compress)
	logger.Infof("[flb-go %d] plugin endpoint parameter = '%s'", operatorID, endpoint)
	logger.Infof("[flb-go %d] plugin autoCreateBucket parameter = '%s'", operatorID, autoCreateBucket)
	logger.Infof("[flb-go %d] plugin timeZone parameter = '%s'", operatorID, timeZone)


	cfg := aws.Config{
		Region:      config.region,
	}
	if config.credentials != nil {
		cfg.WithCredentials(config.credentials)
	}
	if config.endpoint != "" {
		cfg.WithEndpoint(config.endpoint).WithS3ForcePathStyle(true)
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: cfg,
		SharedConfigState: session.SharedConfigEnable,
	}))

	if config.autoCreateBucket == true {
		_, err = ensureBucket(sess, config.bucket, config.region)
		if err != nil {
			return nil, err
		}
	}

	uploader := s3manager.NewUploader(sess, func(u *s3manager.Uploader) {
		u.PartSize = 5 * 1024 * 1024
		u.LeavePartsOnError = true
	})

	s3operator := &s3operator{
		bucket:          *config.bucket,
		prefix:          *config.s3prefix,
		suffixAlgorithm: config.suffixAlgorithm,
		uploader:        uploader,
		compressFormat:  config.compress,
		logger:          logger,
		timeFormat:      config.timeFormat,
		location:        config.location,
	}

	return s3operator, nil

}

func addS3Output(ctx unsafe.Pointer) error {
	operatorID := len(s3operators)
	logger.Infof("[s3operator] id = %d", operatorID)
	// Set the context to point to any Go variable
	context.PluginSetContext(ctx, operatorID)
	operator, err := newS3Output(ctx, operatorID)
	if err != nil {
		return err
	}

	s3operators = append(s3operators, operator)
	return nil
}

func getS3Operator(ctx unsafe.Pointer) *s3operator {
	operatorID := context.PluginGetContext(ctx).(int)
	return s3operators[operatorID]
}

//export FLBPluginInit
// (fluentbit will call this)
// ctx (context) pointer to fluentbit context (state/ c code)
func FLBPluginInit(ctx unsafe.Pointer) int {
	err := addS3Output(ctx)
	if err != nil {
		logger.Infof("Error: %s", err)
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
			s3operator.logger.Warnf("error creating message for S3: %v", err)
			continue
		}
		lines += line + "\n"
	}

	objectKey := GenerateObjectKey(s3operator, time.Now(), lines)
	err := plugin.Put(s3operator, objectKey, time.Now(), lines)
	if err != nil {
		s3operator.logger.Warnf("error sending message for S3: %v", err)
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
func GenerateObjectKey(s3operator *s3operator, t time.Time, lines string) string {
	var fileext string
	switch s3operator.compressFormat {
	case plainTextFormat:
		fileext = ".log"
	case gzipFormat:
		fileext = ".log.gz"
	}
	var suffix string
	switch s3operator.suffixAlgorithm {
	case noSuffixAlgorithm:
		suffix = ""
	case sha256SuffixAlgorithm:
		b := sha256.Sum256([]byte(lines))
		suffix = fmt.Sprintf("-%s", hex.EncodeToString(b[:]))
	}
	// Convert time.Time object's Local with specified TimeZone's
	time.Local = s3operator.location
	timestamp := t.Local().Format("20060102150405")

	fileName := strings.Join([]string{timestamp, suffix, fileext}, "")

	objectKey := filepath.Join(s3operator.prefix, t.Local().Format(s3operator.timeFormat), fileName)
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

func obfuscateSecret(message string) string {
	res := ""
	msgLen := len(message)
	if msgLen > 0 {
		if msgLen >= 3 {
			res = message[:1] + "..." + message[msgLen-1:]
		} else if msgLen < 3 {
			res = message[:1] + "..."
		}
	}

	return res
}

func main() {
}
