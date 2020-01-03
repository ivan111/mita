.PHONY: all
all: test build

.PHONY: build
build:
	go build -ldflags="-s"

.PHONY: test
test:
	go test -cover
