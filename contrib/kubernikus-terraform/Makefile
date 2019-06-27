SHELL    := /bin/sh
IMAGE    := sapcc/kubernikus-terraform
DATE     := $(shell date +%Y%m%d%H%M%S)
VERSION  ?= v$(DATE)

TERRAFORM_VERSION                    := 0.11.7-r0
TERRAFORM_PROVIDER_OPENSTACK_VERSION := b2c32b19aeb26e10a1d1a1d331f89ba2fdc4ef53
TERRAFORM_PROVIDER_CCLOUD_VERSION    := acfba74f111225c9af5b78bb6a93e7f6727eb7f5

.PHONY: all
all: build push

build: 
	docker build -t $(IMAGE):$(VERSION) -t $(IMAGE):latest \
  --build-arg TERRAFORM_VERSION=$(TERRAFORM_VERSION) \
  --build-arg TERRAFORM_PROVIDER_OPENSTACK_VERSION=$(TERRAFORM_PROVIDER_OPENSTACK_VERSION) \
  --build-arg TERRAFORM_PROVIDER_CCLOUD_VERSION=$(TERRAFORM_PROVIDER_CCLOUD_VERSION) \
	.

push:
	docker push $(IMAGE):$(VERSION)   
	docker push $(IMAGE):latest
