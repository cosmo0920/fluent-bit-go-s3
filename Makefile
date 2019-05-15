all: test
	go build -buildmode=c-shared -o out_s3.so .

fast:
	go build out_s3.go s3.go

test:
	go test -cover -race -coverprofile=coverage.txt -covermode=atomic

dep:
	dep ensure

clean:
	rm -rf *.so *.h
