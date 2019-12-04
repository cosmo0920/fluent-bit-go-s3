ifeq ($(OS),Windows_NT)
    DLLEXT := .dll
    TEST_OPTS := ./... -v
else
    DLLEXT := .so
    TEST_OPTS := -cover -race -coverprofile=coverage.txt -covermode=atomic
endif

GO_FLAGS := -tags netgo

all: test build
build:
	go build $(GO_FLAGS) -buildmode=c-shared -o out_s3.so .

fast:
	go build out_s3.go s3.go

test:
	go test $(TEST_OPTS)

dep:
	dep ensure

clean:
	rm -rf *.so *.h
