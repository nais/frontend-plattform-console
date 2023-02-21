BUILDTIME = $(shell date "+%s")
DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git rev-parse --short HEAD)
LDFLAGS := -X github.com/nais/bifrost/pkg/version.Revision=$(LAST_COMMIT) -X github.com/nais/bifrost/pkg/version.Date=$(DATE) -X github.com/nais/bifrost/pkg/version.BuildUnixTime=$(BUILDTIME)

.PHONY: alpine bifrost test

all: fmt check test bifrost

bifrost:
	go build -o bin/bifrost -ldflags "-s $(LDFLAGS)" .

test:
	go test ./...

fmt:
	go run mvdan.cc/gofumpt -w ./

check:
	go run honnef.co/go/tools/cmd/staticcheck ./...
	go run golang.org/x/vuln/cmd/govulncheck -v ./...

alpine:
	go build -a -installsuffix cgo -o bin/bifrost -ldflags "-s $(LDFLAGS)" .

docker:
	docker build -t ghcr.io/nais/bifrost:latest .
