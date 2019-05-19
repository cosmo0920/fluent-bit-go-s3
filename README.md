# fluent-bit s3 output plugin

[![Build Status](https://travis-ci.org/cosmo0920/fluent-bit-go-s3.svg?branch=master)](https://travis-ci.org/cosmo0920/fluent-bit-go-s3)
[![Build status](https://ci.appveyor.com/api/projects/status/93vh3rocl4yxcmg6/branch/master?svg=true)](https://ci.appveyor.com/project/cosmo0920/fluent-bit-go-s3/branch/master)

This plugin works with fluent-bit's go plugin interface. You can use fluent-bit-go-s3 to ship logs into AWS S3.

The configuration typically looks like:

```graphviz
fluent-bit --> AWS S3
```

# Usage

```bash
$ fluent-bit -e /path/to/built/out_s3.so -c fluent-bit.conf
```

# Prerequisites

* Go 1.11+
* gcc (for cgo)

## Building

```bash
$ make
```

### Configuration Options

| Key             | Description                   | Default value | Note                            |
|-----------------|-------------------------------|---------------|---------------------------------|
| Credential      | URI of AWS shared credential  | `""`          |(See [Credentials](#credentials))|
| AccessKeyID     | Access key ID of AWS          | `""`          |(See [Credentials](#credentials))|
| SecretAccessKey | Secret access key ID of AWS   | `""`          |(See [Credentials](#credentials))|
| Bucket          | Bucket name of S3 storage     | `-`           | Mandatory parameter             |
| S3Prefix        | S3Prefix of S3 key            | `-`           | Mandatory parameter             |
| Region          | Region of S3                  | `-`           | Mandatory parameter             |

Example:

add this section to fluent-bit.conf

```properties
[Output]
    Name s3
    Match *
    # Credential    /path/to/sharedcredentialfile
    AccessKeyID     yourawsaccesskeyid
    SecretAccessKey yourawssecretaccesskey
    Bucket          yourbucketname
    S3Prefix yours3prefixname
    S3Region us-east-1
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

And specify the following parameter in fluent-bit configuration:

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
