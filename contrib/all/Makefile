VERSION?=v1.25.3
VERSION_CCM?=v1.25.3
REGISTRY?=registry.k8s.io
IMAGE?=keppel.eu-de-1.cloud.sap/ccloud
ARCH?=amd64

# see https://github.com/kubernetes/release/blob/master/images/build/debian-base/variants.yaml
BASE_VERSION?=bullseye-v1.4.1
BASE_IMAGE?=registry.k8s.io/build-image/debian-base:${BASE_VERSION}

CNI_PLUGINS_RELEASE=v1.0.1

CSI_ATTACHER_VERSION=v3.5.0
CSI_PROVISIONER_VERSION=v3.2.1
CSI_SNAPSHOTTER_VERSION=v5.0.1
CSI_SNAPSHOT_CONTROLLER_VERSION=v5.0.1
CSI_RESIZER_VERSION=v1.5.0
CSI_LIVENESSPROBE_VERSION=v2.7.0
CSI_NODE_DRIVER_VERSION=v2.5.1
CSI_PLUGIN_VERSION?=v1.25.2

ETCD_VERSION=v3.4.13-bootstrap-3
ETCDBRCTL_VERSION=v0.15.4
FLANNEL_VERSION=v0.17.0
COREDNS_VERSION=1.9.1

OPTS?=--network=host

all: build build-csi push push-csi pull

build:
	docker build ${OPTS} -t ${IMAGE}/kube-apiserver:${VERSION} --build-arg IMAGE=${REGISTRY}/kube-apiserver:${VERSION} -f Dockerfile.apiserver .
	docker build ${OPTS} -t ${IMAGE}/kube-scheduler:${VERSION} --build-arg IMAGE=${REGISTRY}/kube-scheduler:${VERSION} .
	docker build ${OPTS} -t ${IMAGE}/kube-controller-manager:${VERSION} --build-arg IMAGE=${REGISTRY}/kube-controller-manager:${VERSION} .
	docker build ${OPTS} -t ${IMAGE}/kube-proxy:${VERSION} --build-arg IMAGE=${REGISTRY}/kube-proxy:${VERSION} .
	docker build ${OPTS} -t ${IMAGE}/kubelet:${VERSION} --build-arg BASE_IMAGE=${BASE_IMAGE} --build-arg KUBERNETES_VERSION=${VERSION} --build-arg ARCH=${ARCH} --build-arg CNI_PLUGINS_RELEASE=${CNI_PLUGINS_RELEASE} - < Dockerfile.kubelet
	docker build ${OPTS} -t ${IMAGE}/openstack-cloud-controller-manager:${VERSION_CCM} --build-arg IMAGE=k8scloudprovider/openstack-cloud-controller-manager:${VERSION_CCM} .
	docker build ${OPTS} -t ${IMAGE}/etcd:${ETCD_VERSION} --build-arg IMAGE=eu.gcr.io/gardener-project/gardener/etcd:${ETCD_VERSION} .
	docker build ${OPTS} -t ${IMAGE}/etcdbrctl:${ETCDBRCTL_VERSION} --build-arg IMAGE=eu.gcr.io/gardener-project/gardener/etcdbrctl:${ETCDBRCTL_VERSION} .

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
	docker push ${IMAGE}/etcd:${ETCD_VERSION}
	docker push ${IMAGE}/etcdbrctl:${ETCDBRCTL_VERSION}

push-csi:
	docker push ${IMAGE}/csi-attacher:${CSI_ATTACHER_VERSION}
	docker push ${IMAGE}/csi-provisioner:${CSI_PROVISIONER_VERSION}
	docker push ${IMAGE}/csi-snapshotter:${CSI_SNAPSHOTTER_VERSION}
	docker push ${IMAGE}/csi-snapshot-controller:${CSI_SNAPSHOT_CONTROLLER_VERSION}
	docker push ${IMAGE}/csi-resizer:${CSI_RESIZER_VERSION}
	docker push ${IMAGE}/csi-livenessprobe:${CSI_LIVENESSPROBE_VERSION}
	docker push ${IMAGE}/csi-node-driver-registrar:${CSI_NODE_DRIVER_VERSION}
	docker push ${IMAGE}/cinder-csi-plugin:${CSI_PLUGIN_VERSION}

pull:
	docker pull keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/flannelcni/flannel:${FLANNEL_VERSION}
	docker pull keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/coredns/coredns:${COREDNS_VERSION}

.PHONY: all build build-csi push push-csi pull
