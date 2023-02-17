VERSION?=v1.25.3
REGISTRY?=registry.k8s.io
IMAGE?=keppel.eu-de-1.cloud.sap/ccloud
ARCH?=amd64

# see https://github.com/kubernetes/release/blob/master/images/build/debian-base/variants.yaml
BASE_VERSION?=bullseye-v1.4.1
BASE_IMAGE?=registry.k8s.io/build-image/debian-base:${BASE_VERSION}

CNI_PLUGINS_RELEASE=v1.0.1

FLANNEL_VERSION=v0.17.0
COREDNS_VERSION=1.9.1

OPTS?=--network=host

all: build push

build:
	docker build ${OPTS} -t ${IMAGE}/kube-apiserver:${VERSION} --build-arg IMAGE=${REGISTRY}/kube-apiserver:${VERSION} -f Dockerfile.apiserver .
	docker build ${OPTS} -t ${IMAGE}/kubelet:${VERSION} --build-arg BASE_IMAGE=${BASE_IMAGE} --build-arg KUBERNETES_VERSION=${VERSION} --build-arg ARCH=${ARCH} --build-arg CNI_PLUGINS_RELEASE=${CNI_PLUGINS_RELEASE} - < Dockerfile.kubelet

push:
	docker push ${IMAGE}/kube-apiserver:${VERSION}
	docker push ${IMAGE}/kubelet:${VERSION}

.PHONY: all build push
