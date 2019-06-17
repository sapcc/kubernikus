IMAGE?=sapcc/hyperkube

ARCH=amd64
CNI_PLUGINS_RELEASE=v0.8.1
CNI_DEDUP_BRIDGE_RELEASE=v0.1.0
MULTUS_RELEASE=3.2
BASEIMAGE?=k8s.gcr.io/hyperkube-$(ARCH):$(VERSION)

TEMP_DIR:=$(shell mktemp -d -t hyperkubeXXXXXX)

build: cni
ifndef VERSION
    $(error VERSION is undefined)
endif
	cp -r ./* ${TEMP_DIR}
	cd ${TEMP_DIR} && sed -i.back "s|BASEIMAGE|${BASEIMAGE}|g" Dockerfile
	docker build --pull -t ${IMAGE}:${VERSION} ${TEMP_DIR}
	rm -rf "${TEMP_DIR}"

push: build
	docker push ${IMAGE}:${VERSION} 

cni:
	mkdir -p ${TEMP_DIR}/cni-bin/bin
	curl -sSL --retry 5 https://github.com/containernetworking/plugins/releases/download/${CNI_PLUGINS_RELEASE}/cni-plugins-linux-${ARCH}-${CNI_PLUGINS_RELEASE}.tgz | tar -xz -C ${TEMP_DIR}/cni-bin/bin
	curl -sSL --retry 5 https://github.com/intel/multus-cni/releases/download/v${MULTUS_RELEASE}/multus-cni_${MULTUS_RELEASE}_linux_amd64.tar.gz | tar --strip 1 -xz -C ${TEMP_DIR}/cni-bin/bin
	curl -o ${TEMP_DIR}/cni-bin/bin/dedup-bridge -sSL --retry 5 https://github.com/sapcc/cni-dedup-bridge/releases/download/${CNI_DEDUP_BRIDGE_RELEASE}/dedup-bridge 
	chmod +x ${TEMP_DIR}/cni-bin/bin/dedup-bridge

.PHONY: build push all cni
