FROM golang:1.14.0-buster AS build-env
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64
ENV GOPATH=/go
ADD . /go/src/github.com/cosmo0920/fluent-bit-go-s3
WORKDIR /go/src/github.com/cosmo0920/fluent-bit-go-s3
RUN make build

FROM fluent/fluent-bit:1.5.4
LABEL Description="Fluent Bit Go S3" FluentBitVersion="1.5.4"

MAINTAINER Hiroshi Hatake <cosmo0920.wp[at]gmail.com>
COPY --from=build-env /go/src/github.com/cosmo0920/fluent-bit-go-s3/out_s3.so /usr/lib/x86_64-linux-gnu/
COPY docker/fluent-bit-s3.conf \
     /fluent-bit/etc/
EXPOSE 2020

# Entry point
CMD ["/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/etc/fluent-bit-s3.conf", "-e", "/usr/lib/x86_64-linux-gnu/out_s3.so"]
