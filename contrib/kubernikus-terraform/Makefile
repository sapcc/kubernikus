SHELL    := /bin/sh
IMAGE    := sapcc/kubernikus-terrraform
DATE     := $(shell date +%Y%m%d%H%M%S)
VERSION  ?= v$(DATE)

TERRAFORM_VERSION                    := 0.11.7-r0
TERRAFORM_PROVIDER_OPENSTACK_VERSION := 8a72ef5435ea15092061fc8acad756a9e87bf93a
TERRAFORM_PROVIDER_CCLOUD_VERSION    := 59c10debdb5576da5b8e2dcd66a815c376e68e08

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
