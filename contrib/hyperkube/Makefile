IMAGE?=sapcc/hyperkube

ARCH=amd64
CNI_RELEASE=v0.6.0
CNI_PLUGINS_RELEASE=v0.7.1
BASEIMAGE=k8s.gcr.io/hyperkube-$(ARCH):$(VERSION)
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
	curl -sSL --retry 5 https://github.com/containernetworking/cni/releases/download/${CNI_RELEASE}/cni-${ARCH}-${CNI_RELEASE}.tgz | tar -xz -C ${TEMP_DIR}/cni-bin/bin
	curl -sSL --retry 5 https://github.com/containernetworking/plugins/releases/download/${CNI_PLUGINS_RELEASE}/cni-plugins-${ARCH}-${CNI_PLUGINS_RELEASE}.tgz | tar -xz -C ${TEMP_DIR}/cni-bin/bin

.PHONY: build push all cni
