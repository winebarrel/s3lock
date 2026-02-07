.PHONY: all
all: vet test build

.PHONY: build
build:
		# TODO:
		# go build ./cmd/s3lock

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -v -count=1 ./...

.PHONY: lint
lint:
	golangci-lint run
