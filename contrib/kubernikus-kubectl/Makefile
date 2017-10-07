SHELL    := /bin/sh
IMAGE    := sapcc/kubernikus-kubectl

KUBERNETES_VERSIONS := v1.7.7 v1.8.0
HELM_VERSION        := v2.6.1

.PHONY: all
all: build push

.PHONY: build 
build: $(KUBERNETES_VERSIONS:%=build-%)

.PHONY: push 
push: $(KUBERNETES_VERSIONS:%=push-%)

build-%:
	docker build -t $(IMAGE):$* --rm \
  --build-arg KUBERNETES_VERSION=$* \
  --build-arg HELM_VERSION=$(HELM_VERSION) \
	.
push-%:
	docker push $(IMAGE):$*

