DATE     = $(shell date +%Y%m%d%H%M)
IMAGE    ?= sapcc/kubernikus
VERSION  ?= v$(DATE)
GOOS     ?= $(shell go env | grep GOOS | cut -d'"' -f2)
BINARIES := groundctl apiserver

LDFLAGS := -X github.com/sapcc/kubernikus/pkg/version.VERSION=$(VERSION)
GOFLAGS := -ldflags "$(LDFLAGS)"

SRCDIRS  := pkg cmd
PACKAGES := $(shell find $(SRCDIRS) -type d)
GOFILES  := $(addsuffix /*.go,$(PACKAGES))
GOFILES  := $(wildcard $(GOFILES))

BUILD_ARGS = --build-arg VERSION=$(VERSION)

ifneq ($(http_proxy),)
BUILD_ARGS+= --build-arg http_proxy=$(http_proxy) --build-arg https_proxy=$(https_proxy) --build-arg no_proxy=$(no_proxy)
endif

HAS_GLIDE := $(shell command -v glide;)
HAS_SWAGGER := $(shell command -v swagger;)

.PHONY: all clean

all: $(BINARIES:%=bin/$(GOOS)/%)

bin/%: $(GOFILES) Makefile
	GOOS=$(*D) GOARCH=amd64 go build $(GOFLAGS) -v -i -o $(@D)/$(@F) ./cmd/$(@F)

build: $(BINARIES:bin/linux/%)
	docker build $(BUILD_ARGS) -t $(IMAGE):$(VERSION) .

push:
	docker push $(IMAGE):$(VERSION)

pkg/api/rest/operations/kubernikus_api.go: swagger.yml
ifndef HAS_SWAGGER
	$(error You need to have go-swagger installed. Run make bootstrap to fix.)
endif
	swagger generate server --name kubernikus --target pkg/api --model-package models \
		--server-package rest --flag-strategy pflag --principal models.Principal --exclude-main

swagger-generate:
	make -B pkg/api/rest/operations/kubernikus_api.go

clean:
	rm -rf bin/*

bootstrap:
ifndef HAS_GLIDE
	brew install glide
endif
ifndef HAS_SWAGGER
	brew install go-swagger
endif
