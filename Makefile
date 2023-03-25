# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/nais/bifrost:main

BUILDTIME = $(shell date "+%s")
DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git rev-parse --short HEAD)
LDFLAGS := -X github.com/nais/bifrost/pkg/version.Revision=$(LAST_COMMIT) -X github.com/nais/bifrost/pkg/version.Date=$(DATE) -X github.com/nais/bifrost/pkg/version.BuildUnixTime=$(BUILDTIME)

.PHONY: all
all: fmt check test bifrost

.PHONY: bifrost
bifrost:
	go build -o bin/bifrost -ldflags "-s $(LDFLAGS)" .

.PHONY: test
test:
	go test ./...

.PHONY: fmt
fmt: gofumpt
	$(GOFUMPT) -w ./

.PHONY: check
check: staticcheck govulncheck
	$(STATICCHECK) ./...
	$(GOVULNCHECK) -v ./...

.PHONY: alpine
alpine:
	go build -a -installsuffix cgo -o bin/bifrost -ldflags "-s $(LDFLAGS)" .

.PHONY: docker
docker:
	docker build -t ${IMG} .

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOVULNCHECK ?= $(LOCALBIN)/govulncheck
STATICCHECK ?= $(LOCALBIN)/staticcheck
GOFUMPT ?= $(LOCALBIN)/gofumpt

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK) ## Download govulncheck locally if necessary.
$(GOVULNCHECK): $(LOCALBIN)
	test -s $(LOCALBIN)/govulncheck || GOBIN=$(LOCALBIN) go install golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: staticcheck
staticcheck: $(STATICCHECK) ## Download staticcheck locally if necessary.
$(STATICCHECK): $(LOCALBIN)
	test -s $(LOCALBIN)/staticcheck || GOBIN=$(LOCALBIN) go install honnef.co/go/tools/cmd/staticcheck@latest

.PHONY: gofumpt
gofumpt: $(GOFUMPT) ## Download gofumpt locally if necessary.
$(GOFUMPT): $(LOCALBIN)
	test -s $(LOCALBIN)/gofumpt || GOBIN=$(LOCALBIN) go install mvdan.cc/gofumpt@latest
