VERSION?=v1.23.6
REGISTRY?=k8s.gcr.io
REPOSITORY?=kubelet-bm
IMAGE?=keppel.eu-de-1.cloud.sap/ccloud
ARCH=amd64

CNI_PLUGINS_RELEASE=v1.1.1
CNI_DEDUP_BRIDGE_RELEASE=v0.1.0
MULTUS_RELEASE=3.8

OPTS?=

all: build push

build:
	docker build ${OPTS} -t ${IMAGE}/${REPOSITORY}:${VERSION} --build-arg KUBERNETES_VERSION=${VERSION} --build-arg ARCH=${ARCH} --build-arg CNI_PLUGINS_RELEASE=${CNI_PLUGINS_RELEASE} --build-arg CNI_DEDUP_BRIDGE_RELEASE=${CNI_DEDUP_BRIDGE_RELEASE} --build-arg MULTUS_RELEASE=${MULTUS_RELEASE} .

push:
	docker push ${IMAGE}/kubelet-bm:${VERSION}

mac:
	docker buildx build --platform linux/amd64 ${OPTS} -t ${IMAGE}/${REPOSITORY}:${VERSION} --build-arg KUBERNETES_VERSION=${VERSION} --build-arg ARCH=${ARCH} --build-arg CNI_PLUGINS_RELEASE=${CNI_PLUGINS_RELEASE} --build-arg CNI_DEDUP_BRIDGE_RELEASE=${CNI_DEDUP_BRIDGE_RELEASE} --build-arg MULTUS_RELEASE=${MULTUS_RELEASE} . --push

.PHONY: all build push
