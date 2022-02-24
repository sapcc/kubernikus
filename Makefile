VERSION ?= $(shell git rev-parse --verify HEAD)
GOOS    ?= $(shell go env GOOS)
GOARCH  ?= $(shell go env GOARCH)
ifeq ($(GOOS),darwin)
export CGO_ENABLED=0
endif
BINARIES := apiserver kubernikus kubernikusctl wormhole

LDFLAGS := -X github.com/sapcc/kubernikus/pkg/version.GitCommit=$(VERSION)
GOFLAGS := -ldflags "$(LDFLAGS) -s -w"

SRCDIRS  := pkg cmd
PACKAGES := $(shell find $(SRCDIRS) -type d)
GOFILES  := $(addsuffix /*.go,$(PACKAGES))
GOFILES  := $(wildcard $(GOFILES))

BUILD_ARGS = --build-arg VERSION=$(VERSION)

ifneq ($(http_proxy),)
BUILD_ARGS+= --build-arg http_proxy=$(http_proxy) --build-arg https_proxy=$(https_proxy) --build-arg no_proxy=$(no_proxy)
endif

GO_SWAGGER_VERSION := v0.18.0
SWAGGER_BIN        ?= bin/$(GOOS)/swagger-$(GO_SWAGGER_VERSION)

.PHONY: all test clean code-gen vendor

all: $(BINARIES:%=bin/$(GOOS)/%)

bin/$(GOOS)/swagger-%:
	curl -f --create-dirs -z $@ -o $@ -L'#' https://github.com/go-swagger/go-swagger/releases/download/$*/swagger_$(GOOS)_amd64
	chmod +x $@

bin/%: $(GOFILES) Makefile
	GOOS=$(*D) GOARCH=$(GOARCH) go build $(GOFLAGS) -v -o $(@D)/$(@F) ./cmd/$(basename $(@F))

test: linters gotest build-e2e

gofmt:
	test/gofmt.sh pkg/ cmd/ test/

linters:
	golangci-lint run -v

gotest:
	# go 1.11 requires gcc for go test because of reasons: https://github.com/golang/go/issues/28065 (CGO_ENABLED=0 fixes this)
	set -o pipefail && CGO_ENABLED=0 go test -v github.com/sapcc/kubernikus/pkg... github.com/sapcc/kubernikus/cmd/... | grep -v 'no test files'

build:
	docker build $(BUILD_ARGS) -t sapcc/kubernikus-binaries:$(VERSION)     -f Dockerfile.kubernikus-binaries .
	docker build $(BUILD_ARGS) -t sapcc/kubernikus-docs-builder:$(VERSION) --cache-from=sapcc/kubernikus-docs-builder:latest ./contrib/kubernikus-docs-builder
	docker build $(BUILD_ARGS) -t sapcc/kubernikusctl:$(VERSION)                                                             ./contrib/kubernikusctl
	docker build $(BUILD_ARGS) -t sapcc/kubernikus-docs:$(VERSION)         -f Dockerfile.kubernikus-docs .
	docker build $(BUILD_ARGS) -t sapcc/kubernikus:$(VERSION)              -f Dockerfile .

pull:
	docker pull sapcc/kubernikus-docs-builder:latest

tag:
	docker tag sapcc/kubernikus:$(VERSION)         sapcc/kubernikus:latest
	docker tag sapcc/kubernikusctl:$(VERSION)      sapcc/kubernikusctl:latest

push:
	docker push sapcc/kubernikus:$(VERSION)
	docker push sapcc/kubernikus:latest
	docker push sapcc/kubernikusctl:$(VERSION)
	docker push sapcc/kubernikusctl:latest

CHANGELOG.md:
ifndef GITHUB_TOKEN
	$(error set GITHUB_TOKEN to a personal access token that has repo:read permission)
else
	docker build $(BUILD_ARGS) -t sapcc/kubernikus-changelog-builder:$(VERSION) --cache-from=sapcc/kubernikus-changelog-builder:latest ./contrib/kubernikus-changelog-builder
	docker tag sapcc/kubernikus-changelog-builder:$(VERSION)  sapcc/kubernikus-changelog-builder:latest
	docker run --rm -v $(PWD):/host -e GITHUB_TOKEN=$(GITHUB_TOKEN) sapcc/kubernikus-changelog-builder:latest
endif

documentation:
	docker build $(BUILD_ARGS) -t sapcc/kubernikus-docs-builder:$(VERSION) --cache-from=sapcc/kubernikus-docs-builder:latest ./contrib/kubernikus-docs-builder
	docker build $(BUILD_ARGS) -t sapcc/kubernikus-docs:$(VERSION)         -f Dockerfile.kubernikus-docs .
	docker tag sapcc/kubernikus-docs:$(VERSION)  sapcc/kubernikus-docs:latest

gh-pages:
	docker run --name gh-pages sapcc/kubernikus-docs:$(VERSION) /bin/true
	docker cp gh-pages:/public/kubernikus gh-pages
	docker rm gh-pages

tests-image:
	docker build $(BUILD_ARGS) -t sapcc/kubernikus-tests:$(VERSION) --cache-from=sapcc/kubernikus-tests:latest ./contrib/kubernikus-tests
	docker tag sapcc/kubernikus-tests:$(VERSION) sapcc/kubernikus-tests:latest
	docker push sapcc/kubernikus-tests:$(VERSION)
	docker push sapcc/kubernikus-tests:latest

pkg/api/rest/operations/kubernikus_api.go: swagger.yml
ifneq (,$(wildcard $(SWAGGER_BIN)))
	$(SWAGGER_BIN) generate server --name kubernikus --target pkg/api --model-package models \
		--server-package rest --flag-strategy pflag --principal models.Principal --exclude-main \
		--with-flatten=full --with-flatten=verbose
	sed -i.foo -e 's/int64 `json:"\([^,]*\),omitempty"`/int64 `json:"\1"`/' pkg/api/models/*.go
	rm -f pkg/api/models/*.go.foo
	sed -e's/^package.*/package spec/' pkg/api/rest/embedded_spec.go > pkg/api/spec/embedded_spec.go
	rm pkg/api/rest/embedded_spec.go
else
	$(warning WARNING: $(SWAGGER_BIN) missing. Run `make bootstrap` to fix.)
endif


swagger-generate:
	make -B pkg/api/rest/operations/kubernikus_api.go

pkg/api/client/kubernikus_client.go: swagger.yml
ifneq (,$(wildcard $(SWAGGER_BIN)))
	$(SWAGGER_BIN) generate client --name kubernikus --target pkg/api --client-package client \
		--existing-models github.com/sapcc/kubernikus/pkg/api/models \
		--skip-models \
		--default-scheme=https --with-flatten=full --with-flatten=verbose \
		--principal models.Principal
else
	$(warning WARNING: $(SWAGGER_BIN) missing. Run `make bootstrap` to fix.)
endif

swagger-generate-client:
	make -B pkg/api/client/kubernikus_client.go

clean:
	rm -rf bin/*

.PHONY: build-e2e
build-e2e:
	CGO_ENABLED=0 go test -v -c -o /dev/null ./test/e2e

.PHONY: test-e2e
test-e2e:
ifdef KS_PASSWORD
	export OS_PASSWORD=$(KS_PASSWORD)
endif
ifndef KUBERNIKUS_URL
	$(error set KUBERNIKUS_URL)
else
	@cd test/e2e && \
	set -o pipefail && \
	go test -v -timeout 55m --kubernikus=$(KUBERNIKUS_URL) | \
	grep -v "CONT\|PAUSE"
endif

.PHONY: test-charts
test-charts:
	docker run --rm -v $(shell pwd):/go/src/github.com/sapcc/kubernikus --entrypoint "/go/src/github.com/sapcc/kubernikus/test/charts/charts.sh" sapcc/kubernikus-tests:latest

.PHONY: test-loopref
test-loopref:
	docker run --rm -v $(shell pwd):/go/src/github.com/sapcc/kubernikus -e "CGO_ENABLED=0" -w /go/src/github.com/sapcc/kubernikus sapcc/kubernikus-tests:latest sh -c "go list ./... | grep "github.com/sapcc/kubernikus/pkg" | xargs exportloopref -c 4"

include code-generate.mk
code-gen: client-gen informer-gen lister-gen deepcopy-gen

vendor:
	go mod vendor -v

bootstrap: $(SWAGGER_BIN)
