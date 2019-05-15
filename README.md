# fluent-bit s3 output plugin

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

| Key             | Description                                   | Default                           |
|-----------------|-----------------------------------------------|-----------------------------------|
| Credential      | URI of AWS shared credential                  | ""                                |
| AccessKeyID     | Access key ID of AWS                          | ""                                |
| SecretAccessKey | Secret access key ID of AWS                   | ""                                |
| Bucket          | Bucket name of S3 storage                     | (specifiying required)            |
| S3Prefix        | S3Prefix of S3 key                            | (specifiying required)            |
| Region          | Region of S3                                  | (specifiying required)            |

Example:

add this section to fluent-bit.conf

```properties
[Output]
    Name s3
    Match *
    AccessKeyID yourawsaccesskeyid
    SecretAccessKey yourawssecretacceddkey
    Bucket yourbucketname
    S3Prefix yours3prefixname
    S3Region us-east-1
```

## Useful links

* [fluent-bit-go](https://github.com/fluent/fluent-bit-go)
