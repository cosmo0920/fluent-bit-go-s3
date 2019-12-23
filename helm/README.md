# Fluent-Bit Go S3 Chart

[Fluent Bit](http://fluentbit.io/) is an open source and multi-platform Log Forwarder.

## Chart Details

This chart will do the following:

* Install a configmap for Fluent Bit with Fluent Bit Go S3 plugin
* Install a deployment that provisions Fluent Bit Go S3 plugin

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
$ helm install --name my-release .
```

## Configuration

The following table lists the configurable parameters of the Fluent-Bit chart and the default values.

**NOTE**: You should use your AWS AccessKeyID and SecretAccessKey in `s3.accessKeyID` and `s3.secretAccessKey`.

| Parameter               | Description                         | Default                 |
| ----------------------- | ----------------------------------- | ----------------------- |
| `s3.accessKeyID`        | Specify AWS AccessKeyID             |                         |
| `s3.secretAccessKey`    | Specify AWS SecretAccessKey         |                         |
| `s3.bucket`             | Specify S3 bucket name              | `fluent-bit-k8s`        |
| `s3.s3prefix`           | Specify S3 prefix name              | `fluent-bit`            |
| `s3.region`             | Specify S3 region                   | `us-east-1`             |
| `s3.compress`           | Whether compress with gzip or not   | `gzip`                  |
| `s3.autoCreateBucket`   | Whether auto creating bucket or not | `true`                  |
| `s3.logLevel`           | Specify logLevel                    | `info`                  |

> **Tip**: You can use the default [values.yaml](values.yaml)
