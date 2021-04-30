VERSION?=v1.21.0
VERSION_CCM?=v1.21.0
REGISTRY?=k8s.gcr.io
IMAGE?=keppel.eu-de-1.cloud.sap/ccloud
ARCH=amd64

#CNI_PLUGINS_RELEASE=v0.8.5

CSI_ATTACHER_VERSION=v2.2.1
CSI_PROVISIONER_VERSION=v2.1.0
CSI_SNAPSHOTTER_VERSION=v2.1.3
CSI_SNAPSHOT_CONTROLLER_VERSION=v2.1.3
CSI_RESIZER_VERSION=v0.5.1
CSI_LIVENESSPROBE_VERSION=v2.1.0
CSI_NODE_DRIVER_VERSION=v1.3.0
CSI_PLUGIN_VERSION?=v1.21.0

OPTS?=

all: build build-csi push push-csi

build:
	docker build ${OPTS} -t ${IMAGE}/kube-apiserver:${VERSION} --build-arg IMAGE=${REGISTRY}/kube-apiserver:${VERSION} -f Dockerfile.apiserver .
	docker build ${OPTS} -t ${IMAGE}/kube-scheduler:${VERSION} --build-arg IMAGE=${REGISTRY}/kube-scheduler:${VERSION} .
	docker build ${OPTS} -t ${IMAGE}/kube-controller-manager:${VERSION} --build-arg IMAGE=${REGISTRY}/kube-controller-manager:${VERSION} .
	docker build ${OPTS} -t ${IMAGE}/kube-proxy:${VERSION} --build-arg IMAGE=${REGISTRY}/kube-proxy:${VERSION} .
	docker build ${OPTS} -t ${IMAGE}/kubelet:${VERSION} --build-arg KUBERNETES_VERSION=${VERSION} --build-arg ARCH=${ARCH} --build-arg CNI_PLUGINS_RELEASE=${CNI_PLUGINS_RELEASE} - < Dockerfile.kubelet
	docker build ${OPTS} -t ${IMAGE}/openstack-cloud-controller-manager:${VERSION_CCM} --build-arg IMAGE=k8scloudprovider/openstack-cloud-controller-manager:${VERSION_CCM} .

build-csi:
	docker build ${OPTS} -t ${IMAGE}/csi-attacher:${CSI_ATTACHER_VERSION} --build-arg IMAGE=${REGISTRY}/sig-storage/csi-attacher:${CSI_ATTACHER_VERSION} .
	docker build ${OPTS} -t ${IMAGE}/csi-provisioner:${CSI_PROVISIONER_VERSION} --build-arg IMAGE=${REGISTRY}/sig-storage/csi-provisioner:${CSI_PROVISIONER_VERSION} .
	docker build ${OPTS} -t ${IMAGE}/csi-snapshotter:${CSI_SNAPSHOTTER_VERSION} --build-arg IMAGE=${REGISTRY}/sig-storage/csi-snapshotter:${CSI_SNAPSHOTTER_VERSION} .
	docker build ${OPTS} -t ${IMAGE}/csi-snapshot-controller:${CSI_SNAPSHOT_CONTROLLER_VERSION} --build-arg IMAGE=${REGISTRY}/sig-storage/snapshot-controller:${CSI_SNAPSHOT_CONTROLLER_VERSION} .
	docker build ${OPTS} -t ${IMAGE}/csi-resizer:${CSI_RESIZER_VERSION} --build-arg IMAGE=${REGISTRY}/sig-storage/csi-resizer:${CSI_RESIZER_VERSION} .
	docker build ${OPTS} -t ${IMAGE}/csi-livenessprobe:${CSI_LIVENESSPROBE_VERSION} --build-arg IMAGE=${REGISTRY}/sig-storage/livenessprobe:${CSI_LIVENESSPROBE_VERSION} .
	docker build ${OPTS} -t ${IMAGE}/csi-node-driver-registrar:${CSI_NODE_DRIVER_VERSION} --build-arg IMAGE=${REGISTRY}/sig-storage/csi-node-driver-registrar:${CSI_NODE_DRIVER_VERSION} .
	docker build ${OPTS} -t ${IMAGE}/cinder-csi-plugin:${CSI_PLUGIN_VERSION} --build-arg IMAGE=docker.io/k8scloudprovider/cinder-csi-plugin:${CSI_PLUGIN_VERSION} .

push:
	docker push ${IMAGE}/kube-apiserver:${VERSION}
	docker push ${IMAGE}/kube-scheduler:${VERSION}
	docker push ${IMAGE}/kube-controller-manager:${VERSION}
	docker push ${IMAGE}/kube-proxy:${VERSION}
	docker push ${IMAGE}/kubelet:${VERSION}
	docker push ${IMAGE}/openstack-cloud-controller-manager:${VERSION_CCM}

push-csi:
	docker push ${IMAGE}/csi-attacher:${CSI_ATTACHER_VERSION}
	docker push ${IMAGE}/csi-provisioner:${CSI_PROVISIONER_VERSION}
	docker push ${IMAGE}/csi-snapshotter:${CSI_SNAPSHOTTER_VERSION}
	docker push ${IMAGE}/csi-snapshot-controller:${CSI_SNAPSHOT_CONTROLLER_VERSION}
	docker push ${IMAGE}/csi-resizer:${CSI_RESIZER_VERSION}
	docker push ${IMAGE}/csi-livenessprobe:${CSI_LIVENESSPROBE_VERSION}
	docker push ${IMAGE}/csi-node-driver-registrar:${CSI_NODE_DRIVER_VERSION}
	docker push ${IMAGE}/cinder-csi-plugin:${CSI_PLUGIN_VERSION}

.PHONY: all build build-csi push push-csi
