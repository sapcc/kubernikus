SHELL    := /bin/sh
DATE     := $(shell date +%Y%m%d%H%M%S)
VERSION  ?= v$(DATE)
IMAGE    := sapcc/kubernikus-changelog-builder

GITHUB_CHANGELOG_GENERATOR_VERSION := 1.14.3
BUILD_ARGS = --build-arg GITHUB_CHANGELOG_GENERATOR_VERSION=$(GITHUB_CHANGELOG_GENERATOR_VERSION)

.PHONY: all
all: build push

build: 
	docker build $(BUILD_ARGS) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

push:
	docker push $(IMAGE):$(VERSION)   
	docker push $(IMAGE):latest
