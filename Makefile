.PHONY: all
all: test build

.PHONY: build
build:
	go generate
	go build -ldflags="-s"

.PHONY: test
test:
	dropdb mita_test
	createdb mita_test
	psql -d mita_test -f public/data/schema.sql
	go test -cover
