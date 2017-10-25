SHELL    := /bin/sh
DATE     := $(shell date +%Y%m%d%H%M%S)
VERSION  ?= v$(DATE)
IMAGE    := sapcc/kubernikus-docs-builder

HUGO_VERSION := 0.30.2  # Without v
BUILD_ARGS = --build-arg HUGO_VERSION=$(HUGO_VERSION)

.PHONY: all
all: build push

build: 
	docker build $(BUILD_ARGS) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

push:
	docker push $(IMAGE):$(VERSION)   
	docker push $(IMAGE):latest
