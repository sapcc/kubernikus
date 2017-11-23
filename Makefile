VERSION  ?= $(shell git rev-parse --verify HEAD)
GOOS     ?= $(shell go env | grep GOOS | cut -d'"' -f2)
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

HAS_GLIDE := $(shell command -v glide;)
HAS_GLIDE_VC := $(shell command -v glide-vc;)
GO_SWAGGER_VERSION := 0.12.0
SWAGGER_BIN        := bin/$(GOOS)/swagger-$(GO_SWAGGER_VERSION)

.PHONY: all clean code-gen client-gen informer-gen lister-gen vendor

all: $(BINARIES:%=bin/$(GOOS)/%)

bin/$(GOOS)/swagger-%:
	curl -f -z $@ -o $@ -L'#' https://github.com/go-swagger/go-swagger/releases/download/$*/swagger_$(GOOS)_amd64
	chmod +x $@

bin/%: $(GOFILES) Makefile
	GOOS=$(*D) GOARCH=amd64 go build $(GOFLAGS) -v -i -o $(@D)/$(@F) ./cmd/$(basename $(@F))

test:
	set -o pipefail && go test -v ./pkg/... ./cmd/... | grep -v 'no test files'

build:
	docker build $(BUILD_ARGS) -t sapcc/kubernikus-binaries:$(VERSION)     -f Dockerfile.kubernikus-binaries .
	docker build $(BUILD_ARGS) -t sapcc/kubernikus-docs-builder:$(VERSION) --cache-from=sapcc/kubernikus-docs-builder:latest ./contrib/kubernikus-docs-builder
	docker build $(BUILD_ARGS) -t sapcc/kubernikus-kubectl:$(VERSION)      --cache-from=sapcc/kubernikus-kubectl:latest      ./contrib/kubernikus-kubectl
	docker build $(BUILD_ARGS) -t sapcc/kubernikusctl:$(VERSION)                                                             ./contrib/kubernikusctl
	docker build $(BUILD_ARGS) -t sapcc/kubernikus-docs:$(VERSION)         -f Dockerfile.kubernikus-docs .
	docker build $(BUILD_ARGS) -t sapcc/kubernikus:$(VERSION)              -f Dockerfile .

pull:
	docker pull sapcc/kubernikus-docs-builder:latest
	docker pull sapcc/kubernikus-kubectl:latest

tag:
	docker tag sapcc/kubernikus:$(VERSION)         sapcc/kubernikus:latest
	docker tag sapcc/kubernikusctl:$(VERSION)      sapcc/kubernikusctl:latest
	docker tag sapcc/kubernikus-kubectl:$(VERSION) sapcc/kubernikus-kubectl:latest

push:
	docker push sapcc/kubernikus:$(VERSION)
	docker push sapcc/kubernikus:latest
	docker push sapcc/kubernikusctl:$(VERSION)
	docker push sapcc/kubernikusctl:latest
	docker push sapcc/kubernikus-kubectl:$(VERSION)
	docker push sapcc/kubernikus-kubectl:latest

gh-pages:
	docker run --name gh-pages sapcc/kubernikus-docs:$(VERSION) /bin/true
	docker cp gh-pages:/public/kubernikus gh-pages
	docker rm gh-pages

pkg/api/rest/operations/kubernikus_api.go: swagger.yml
ifneq (,$(wildcard $(SWAGGER_BIN)))
	$(SWAGGER_BIN) generate server --name kubernikus --target pkg/api --model-package models \
		--server-package rest --flag-strategy pflag --principal models.Principal --exclude-main
	sed -i '' -e 's/int64 `json:"\([^,]*\),omitempty"`/int64 `json:"\1"`/' pkg/api/models/*.go
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
		--principal models.Principal
else
	$(warning WARNING: $(SWAGGER_BIN) missing. Run `make bootstrap` to fix.)
endif

swagger-generate-client:
	make -B pkg/client/kubernikus_generated/kubernikus_client.go

clean:
	rm -rf bin/*

# If the first argument is "test-e2e" the rest is used as an argument to specify which tests phases to execute sequentially
# e.g. make test-e2e create delete
ifeq (test-e2e,$(firstword $(MAKECMDGOALS)))
  ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(addprefix --,$(MAKECMDGOALS)))
endif

.PHONY: test-e2e
test-e2e:
	go run test/e2e/*.go $(ARGS)

include code-generate.mk
code-gen: client-gen informer-gen lister-gen

vendor:
ifndef HAS_GLIDE_VC
	$(error glide-vc (vendor cleaner) not found. Run `make bootstrap to fix.`)
endif
	glide install -v
	glide-vc --only-code --no-tests

bootstrap: $(SWAGGER_BIN)
ifndef HAS_GLIDE
	brew install glide
endif
ifndef HAS_GLIDE_VC
	go get -u github.com/sgotti/glide-vc
endif
