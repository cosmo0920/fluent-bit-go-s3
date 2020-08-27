ifeq ($(OS),Windows_NT)
    DLLEXT := .dll
    TEST_OPTS := ./... -v
else
    DLLEXT := .so
    TEST_OPTS := -cover -race -coverprofile=coverage.txt -covermode=atomic
endif

VERSION := 0.7.2
DOCKER_IMAGE_VERSION := 2.0

# Version info for binaries
GIT_REVISION := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

VPREFIX := github.com/cosmo0920/fluent-bit-go-s3/vendor/github.com/prometheus/common/version
GO_FLAGS := -ldflags "-X $(VPREFIX).Branch=$(GIT_BRANCH) -X $(VPREFIX).Version=$(VERSION) -X $(VPREFIX).Revision=$(GIT_REVISION)" -tags netgo

all: test build
build:
	go build $(GO_FLAGS) -buildmode=c-shared -o out_s3$(DLLEXT) .

fast:
	go build out_s3.go s3.go formatter.go

test:
	go test $(TEST_OPTS)

dep:
	dep ensure

clean:
	rm -rf *$(DLLEXT) *.h

build-image:
	docker build . -t cosmo0920/fluent-bit-go-s3:v$(VERSION)-$(DOCKER_IMAGE_VERSION)

publish-image:
	docker push cosmo0920/fluent-bit-go-s3:v$(VERSION)-$(DOCKER_IMAGE_VERSION)
