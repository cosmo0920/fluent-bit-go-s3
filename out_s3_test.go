package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/fluent/fluent-bit-go/output"
	"github.com/stretchr/testify/assert"
)

func TestCreateJSON(t *testing.T) {
	record := make(map[interface{}]interface{})
	record["key"] = "value"
	record["number"] = 8

	line, err := createJSON(record)
	if err != nil {
		assert.Fail(t, "createJSON fails:%v", err)
	}
	assert.NotNil(t, line, "json string not to be nil")
	result := make(map[string]interface{})
	jsonBytes := ([]byte)(line)
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		assert.Fail(t, "unmarshal of json fails:%v", err)
	}

	assert.Equal(t, result["key"], "value")
	assert.Equal(t, result["number"], float64(8))
}

// ref: https://gist.github.com/ChristopherThorpe/fd3720efe2ba83c929bf4105719ee967
// NestedMapLookup
// m:  a map from strings to other maps or values, of arbitrary depth
// ks: successive keys to reach an internal or leaf node (variadic)
// If an internal node is reached, will return the internal map
//
// Returns: (Exactly one of these will be nil)
// rval: the target node (if found)
// err:  an error created by fmt.Errorf
//
func NestedMapLookup(m map[string]interface{}, ks ...string) (rval interface{}, err error) {
	var ok bool

	if len(ks) == 0 { // degenerate input
		return nil, fmt.Errorf("NestedMapLookup needs at least one key")
	}
	if rval, ok = m[ks[0]]; !ok {
		return nil, fmt.Errorf("key not found; remaining keys: %v", ks)
	} else if len(ks) == 1 { // we've reached the final key
		return rval, nil
	} else if m, ok = rval.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("malformed structure at %#v", rval)
	} else { // 1+ more keys
		return NestedMapLookup(m, ks[1:]...)
	}
}

func TestCreateJSONWithNestedKey(t *testing.T) {
	record := make(map[interface{}]interface{})
	record["key"] = "value"
	record["number"] = 8
	record["nested"] = map[interface{}]interface{}{"key": map[interface{}]interface{}{"key2": "not base64 encoded"}}

	line, err := createJSON(record)
	if err != nil {
		assert.Fail(t, "createJSON fails:%v", err)
	}
	assert.NotNil(t, line, "json string not to be nil")
	result := make(map[string]interface{})
	jsonBytes := ([]byte)(line)
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		assert.Fail(t, "unmarshal of json fails:%v", err)
	}

	assert.Equal(t, result["key"], "value")
	assert.Equal(t, result["number"], float64(8))

	val, err := NestedMapLookup(result, "nested", "key", "key2")
	assert.Equal(t, val, "not base64 encoded")
}

func TestGenerateObjectKey(t *testing.T) {
	now := time.Now()
	s3mock := &s3operator{
		bucket:         "s3examplebucket",
		prefix:         "s3exampleprefix",
		uploader:       nil,
		compressFormat: plainTextFormat,
	}
	lines := "exampletext"
	objectKey := GenerateObjectKey(s3mock, now, lines)
	fmt.Printf("objectKey: %v\n", objectKey)
	assert.NotNil(t, objectKey, "objectKey not to be nil")
}

func TestGenerateObjectKeyWithNoSuffixAlgorithm(t *testing.T) {
	now := time.Now()
	s3mock := &s3operator{
		bucket:          "s3examplebucket",
		prefix:          "s3exampleprefix",
		suffixAlgorithm: noSuffixAlgorithm,
		uploader:        nil,
		compressFormat:  plainTextFormat,
	}
	lines := "exampletext"
	objectKey := GenerateObjectKey(s3mock, now, lines)
	fmt.Printf("objectKey: %v\n", objectKey)
	assert.False(t, strings.HasSuffix(objectKey, "-c675f9cd0e59479e5ccca3ea8a03beccd80f662f6a56662bfc9dd0b61d4f73c3.log"), "objectKey has no suffix")
}

func TestGenerateObjectKeyWithSha256SuffixAlgorithm(t *testing.T) {
	now := time.Now()
	s3mock := &s3operator{
		bucket:          "s3examplebucket",
		prefix:          "s3exampleprefix",
		suffixAlgorithm: sha256SuffixAlgorithm,
		uploader:        nil,
		compressFormat:  plainTextFormat,
	}
	lines := "exampletext"
	objectKey := GenerateObjectKey(s3mock, now, lines)
	fmt.Printf("objectKey: %v\n", objectKey)
	assert.True(t, strings.HasSuffix(objectKey, "-c675f9cd0e59479e5ccca3ea8a03beccd80f662f6a56662bfc9dd0b61d4f73c3.log"), "objectKey has sha256 suffix")
}

func TestGenerateObjectKeyWithTokyoLocation(t *testing.T) {
	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Tokyo")
	s3mock := &s3operator{
		bucket:         "s3examplebucket",
		prefix:         "s3exampleprefix",
		uploader:       nil,
		compressFormat: plainTextFormat,
		location:       loc,
	}
	lines := "exampletext"
	objectKey := GenerateObjectKey(s3mock, now, lines)
	fmt.Printf("objectKey: %v\n", objectKey)
	assert.NotNil(t, objectKey, "objectKey not to be nil")
}

func TestGenerateObjectKeyWithUSEastLocation(t *testing.T) {
	now := time.Now()
	loc, _ := time.LoadLocation("US/Eastern")
	s3mock := &s3operator{
		bucket:         "s3examplebucket",
		prefix:         "s3exampleprefix",
		uploader:       nil,
		compressFormat: plainTextFormat,
		location:       loc,
	}
	lines := "exampletext"
	objectKey := GenerateObjectKey(s3mock, now, lines)
	fmt.Printf("objectKey: %v\n", objectKey)
	assert.NotNil(t, objectKey, "objectKey not to be nil")
}

func TestGenerateObjectKeyWithUTCLocation(t *testing.T) {
	now := time.Now()
	loc, _ := time.LoadLocation("UTC")
	s3mock := &s3operator{
		bucket:         "s3examplebucket",
		prefix:         "s3exampleprefix",
		uploader:       nil,
		compressFormat: plainTextFormat,
		location:       loc,
	}
	lines := "exampletext"
	objectKey := GenerateObjectKey(s3mock, now, lines)
	fmt.Printf("objectKey: %v\n", objectKey)
	assert.NotNil(t, objectKey, "objectKey not to be nil")
}

func TestGenerateObjectKeyWithGzip(t *testing.T) {
	now := time.Now()
	s3mock := &s3operator{
		bucket:         "s3examplebucket",
		prefix:         "s3exampleprefix",
		uploader:       nil,
		compressFormat: gzipFormat,
	}
	lines := "exampletext"
	objectKey := GenerateObjectKey(s3mock, now, lines)
	fmt.Printf("objectKey: %v\n", objectKey)
	assert.NotNil(t, objectKey, "objectKey not to be nil")
}

// based on https://text.baldanders.info/golang/gzip-operation/
func readGzip(dst io.Writer, src io.Reader) error {
	zr, err := gzip.NewReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()

	io.Copy(dst, zr)

	return nil
}

func TestMakeGzip(t *testing.T) {
	var line = "a gzipped string line which is compressed by compress/gzip library written in Go."

	compressed, err := makeGzip([]byte(line))
	if err != nil {
		assert.Fail(t, "compress string with gzip fails:%v", err)
	}

	var b bytes.Buffer
	err = readGzip(&b, bytes.NewReader(compressed))
	if err != nil {
		assert.Fail(t, "decompress from gzippped string fails:%v", err)
	}
	assert.Equal(t, line, b.String())
}

type testrecord struct {
	rc   int
	ts   interface{}
	data map[interface{}]interface{}
}

type events struct {
	data []byte
}
type testFluentPlugin struct {
	credential       string
	accessKeyID      string
	secretAccessKey  string
	bucket           string
	s3prefix         string
	region           string
	compress         string
	endpoint         string
	autoCreateBucket string
	logLevel         string
	location         string
	records          []testrecord
	position         int
	events           []*events
}

func (p *testFluentPlugin) PluginConfigKey(ctx unsafe.Pointer, key string) string {
	switch key {
	case "Credential":
		return p.credential
	case "AccessKeyID":
		return p.accessKeyID
	case "SecretAccessKey":
		return p.secretAccessKey
	case "Bucket":
		return p.bucket
	case "S3Prefix":
		return p.s3prefix
	case "Region":
		return p.region
	case "Compress":
		return p.compress
	case "Endpoint":
		return p.endpoint
	case "AutoCreateBucket":
		return p.autoCreateBucket
	case "LogLevel":
		return p.logLevel
	case "TimeZone":
		return p.location
	}
	return "unknown-" + key
}

func (p *testFluentPlugin) Unregister(ctx unsafe.Pointer) {}
func (p *testFluentPlugin) GetRecord(dec *output.FLBDecoder) (int, interface{}, map[interface{}]interface{}) {
	if p.position < len(p.records) {
		r := p.records[p.position]
		p.position++
		return r.rc, r.ts, r.data
	}
	return -1, nil, nil
}
func (p *testFluentPlugin) NewDecoder(data unsafe.Pointer, length int) *output.FLBDecoder { return nil }
func (p *testFluentPlugin) Exit(code int)                                                 {}
func (p *testFluentPlugin) Put(s3operator *s3operator, objectKey string, timestamp time.Time, line string) error {
	data := ([]byte)(line)
	events := &events{data: data}
	p.events = append(p.events, events)
	return nil
}
func (p *testFluentPlugin) addrecord(rc int, ts interface{}, line map[interface{}]interface{}) {
	p.records = append(p.records, testrecord{rc: rc, ts: ts, data: line})
}

type stubProvider struct {
	creds   credentials.Value
	expired bool
	err     error
}

func (s *stubProvider) Retrieve() (credentials.Value, error) {
	s.expired = false
	s.creds.ProviderName = "stubProvider"
	return s.creds, s.err
}
func (s *stubProvider) IsExpired() bool {
	return s.expired
}

type testS3Credential struct {
	credential string
}

func (c *testS3Credential) GetCredentials(accessID, secretkey, credential string) (*credentials.Credentials, error) {
	creds := credentials.NewCredentials(&stubProvider{
		creds: credentials.Value{
			AccessKeyID:     "AKID",
			SecretAccessKey: "SECRET",
			SessionToken:    "",
		},
		expired: true,
	})

	return creds, nil
}

func TestPluginInitializationWithStaticCredentials(t *testing.T) {
	s3Creds = &testS3Credential{}
	_, err := getS3Config("exampleaccessID", "examplesecretkey", "", "exampleprefix", "", "examplebucket", "exampleregion", "", "", "false", "info", "", "")
	if err != nil {
		t.Fatalf("failed test %#v", err)
	}
	plugin = &testFluentPlugin{
		accessKeyID:      "exampleaccesskeyid",
		secretAccessKey:  "examplesecretaccesskey",
		bucket:           "examplebucket",
		s3prefix:         "exampleprefix",
		region:           "exampleregion",
		compress:         "",
		endpoint:         "",
		autoCreateBucket: "false",
		logLevel:         "info",
	}
	res := FLBPluginInit(unsafe.Pointer(&plugin))
	assert.Equal(t, output.FLB_OK, res)
}

func TestPluginInitializationWithSharedCredentials(t *testing.T) {
	s3Creds = &testS3Credential{}
	_, err := getS3Config("", "", "examplecredentials", "exampleprefix", "", "examplebucket", "exampleregion", "", "", "false", "info", "", "")
	if err != nil {
		t.Fatalf("failed test %#v", err)
	}
	plugin = &testFluentPlugin{
		credential:       "examplecredentials",
		bucket:           "examplebucket",
		s3prefix:         "exampleprefix",
		region:           "exampleregion",
		compress:         "",
		endpoint:         "",
		autoCreateBucket: "false",
		logLevel:         "info",
	}
	res := FLBPluginInit(unsafe.Pointer(&plugin))
	assert.Equal(t, output.FLB_OK, res)
}

func TestPluginFlusher(t *testing.T) {
	testplugin := &testFluentPlugin{
		credential:       "examplecredentials",
		accessKeyID:      "exampleaccesskeyid",
		secretAccessKey:  "examplesecretaccesskey",
		bucket:           "examplebucket",
		s3prefix:         "exampleprefix",
		compress:         "",
		endpoint:         "",
		autoCreateBucket: "false",
	}
	ts := time.Date(2019, time.March, 10, 10, 11, 12, 0, time.UTC)
	testrecords := map[interface{}]interface{}{
		"mykey": "myvalue",
	}
	testplugin.addrecord(0, output.FLBTime{Time: ts}, testrecords)
	testplugin.addrecord(0, uint64(ts.Unix()), testrecords)
	testplugin.addrecord(0, 0, testrecords)
	plugin = testplugin
	res := FLBPluginFlushCtx(nil, nil, 0, nil)
	assert.Equal(t, output.FLB_OK, res)
	assert.Len(t, testplugin.events, 1) // event length should be 1.
	var parsed map[string]interface{}
	json.Unmarshal(testplugin.events[0].data, &parsed)
	expected := `{"mykey":"myvalue"}
{"mykey":"myvalue"}
{"mykey":"myvalue"}
`
	assert.Equal(t, expected, string(testplugin.events[0].data))
}
