.PHONY: all build clean test

.get-deps: *.go
	go get -t -d -v ./...
	go get github.com/vektra/mockery/.../
	touch .get-deps

all: test build handler.zip

build: encrypt53

clean:
	rm -f .get-deps
	rm -f encrypt53
	rm -f handler handler.zip
	rm -rf mocks

mocks:
	rm -rf mocks
	mockery -dir ../../aws/aws-sdk-go/service/route53/*iface -all
	mockery -dir ../../aws/aws-sdk-go/service/s3/*iface -all
	mockery -dir ../../aws/aws-sdk-go/service/sns/*iface -all

test: .get-deps mocks *.go
	go test -v -cover ./...

encrypt53: .get-deps *.go
	go build -o $@ *.go

handler: .get-deps *.go
	GOOS=linux GOARCH=amd64 go build -o $@ *.go

handler.zip: handler
	rm -f handler.zip
	zip handler.zip ./handler

fmt: *.go
	go fmt *.go
