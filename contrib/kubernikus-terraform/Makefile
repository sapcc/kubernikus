SHELL    := /bin/sh
IMAGE    := sapcc/kubernikus-terraform
DATE     := $(shell date +%Y%m%d%H%M%S)
VERSION  ?= v$(DATE)

TERRAFORM_VERSION                    := 0.11.7-r0
TERRAFORM_PROVIDER_OPENSTACK_VERSION := v1.21.0
TERRAFORM_PROVIDER_CCLOUD_VERSION    := v1.1.2


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
