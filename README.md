# fluent-bit s3 output plugin

[![Build Status](https://travis-ci.org/cosmo0920/fluent-bit-go-s3.svg?branch=master)](https://travis-ci.org/cosmo0920/fluent-bit-go-s3)
[![Build status](https://ci.appveyor.com/api/projects/status/93vh3rocl4yxcmg6/branch/master?svg=true)](https://ci.appveyor.com/project/cosmo0920/fluent-bit-go-s3/branch/master)

Windows binaries are available in [release pages](https://github.com/cosmo0920/fluent-bit-go-s3/releases).

This plugin works with fluent-bit's go plugin interface. You can use fluent-bit-go-s3 to ship logs into AWS S3.

The configuration typically looks like:

```graphviz
fluent-bit --> AWS S3
```

# Usage

```bash
$ fluent-bit -e /path/to/built/out_s3.so -c fluent-bit.conf
```

Or,


```bash
$ docker build . -t fluent-bit/s3-plugin
```

and then, specify configuration parameters as environment variables:

```bash
$ docker run -it -e="FLUENT_BIT_ACCESS_KEY_ID=yourawsaccesskey" \
                 -e="FLUENT_BIT_SECRET_ACCESS_KEY=yourawsaccesssecret" \
                 -e="FLUENT_BIT_BUCKET_NAME=yourbucketname" \
                 -e="FLUENT_BIT_S3_PREFIX=yours3prefix" \
                 -e="FLUENT_BIT_REGION=awsregion" \
                 fluent-bit/s3-plugin
```

Using docker image from docker hub.

```bash
$ docker pull cosmo0920/fluent-bit-go-s3:latest
```

Other released images are available in [DockerHub's fluent-bit-go-s3 image tags](https://hub.docker.com/r/cosmo0920/fluent-bit-go-s3/tags).

Or, using helm:

```bash
helm install [YOURRELEASENAME] ./helm/fluent-bit
```

# Prerequisites

* Go 1.11+
* gcc (for cgo)
* make

## Building

```bash
$ make
```

### Configuration Options

| Key              | Description                           | Default value   | Note                                                                 |
|------------------|---------------------------------------|-----------------|----------------------------------------------------------------------|
| Credential       | URI of AWS shared credential          | `""`            | (See [Credentials](#credentials))                                    |
| AccessKeyID      | Access key ID of AWS                  | `""`            | (See [Credentials](#credentials))                                    |
| SecretAccessKey  | Secret access key ID of AWS           | `""`            | (See [Credentials](#credentials))                                    |
| Bucket           | Bucket name of S3 storage             | `-`             | Mandatory parameter                                                  |
| S3Prefix         | S3Prefix of S3 key                    | `-`             | Mandatory parameter                                                  |
| SuffixAlgorithm  | Algorithm for naming S3 object suffix | `""`            | sha256 or no suffix(`""`)                                            |
| Region           | Region of S3                          | `-`             | Mandatory parameter                                                  |
| Compress         | Choose Compress method                | `""`            | gzip or plainText(`""`)                                              |
| Endpoint         | Specify the endpoint URL              | `""`            | URL with port or empty string                                        |
| AutoCreateBucket | Create bucket automatically           | `false`         | true/false                                                           |
| LogLevel         | Specify Log Level                     | `"info"`        | trace/debug/info/warning/error/fatal/panic                           |
| TimeFormat       | Time format to add to the S3 path     | `"20060102/15"` | Specify in [Go's Time Format](https://golang.org/src/time/format.go) | 
| TimeZone         | Specify TimeZone                      | `""`            | Specify TZInfo based region. e.g.) Asia/Tokyo                        |

Example:

Add this section to fluent-bit.conf:

```properties
[Output]
    Name s3
    Match *
    # Credential    /path/to/sharedcredentialfile
    AccessKeyID     yourawsaccesskeyid
    SecretAccessKey yourawssecretaccesskey
    Bucket          yourbucketname
    S3Prefix yours3prefixname
    SuffixAlgorithm sha256
    Region us-east-1
    Compress gzip
    # Endpoint parameter is mainly used for minio.
    # Endpoint http://localhost:9000
    # TimeFormat 20060102/15
    # TimeZone Asia/Tokyo
```

fluent-bit-go-s3 supports the following credentials. Users must specify one of them:

## Credentials

Specifying credentials is **required**.

This plugin supports the following credentials:

### Shared Credentials

Create the following file which includes credentials:

```ini
[default]
aws_access_key_id = YOUR_AWS_ACCESS_KEY_ID
aws_secret_access_key = YOUR_AWS_SECRET_ACCESS_KEY
```

Then, specify the following parameter in fluent-bit configuration:

```ini
Credential    /path/to/sharedcredentialfile
```

### Static Credentials

Specify the following parameters in fluent-bit configuration:

```ini
AccessKeyID     yourawsaccesskeyid
SecretAccessKey yourawssecretaccesskey
```

### Environment Credentials

Specify `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` as environment variables.

## Useful links

* [fluent-bit-go](https://github.com/fluent/fluent-bit-go)
