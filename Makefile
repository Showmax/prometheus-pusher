build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(GOPATH)/bin/prometheus-pusher

#
# Tests-related tasks
#
.PHONY: unit-test
unit-test:
	go test -v .
