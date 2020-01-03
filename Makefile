.PHONY: all
all: test build

.PHONY: build
build:
	go generate
	go build -ldflags="-s"

.PHONY: test
test:
	go test -cover
