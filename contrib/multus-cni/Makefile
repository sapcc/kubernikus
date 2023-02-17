MULTUS_CNI_RELEASE=3.9
IMAGE?=keppel.eu-de-1.cloud.sap/ccloud
OS?=linux
ARCH?=amd64


build:
	docker build -t ${IMAGE}/multus-cni:${MULTUS_CNI_RELEASE} --build-arg MULTUS_CNI_RELEASE=${MULTUS_CNI_RELEASE} .

push:
	docker push ${IMAGE}/multus-cni:${MULTUS_CNI_RELEASE}

mac:
	docker buildx build --platform linux/amd64 -t ${IMAGE}/multus-cni:${MULTUS_CNI_RELEASE} --build-arg MULTUS_CNI_RELEASE=${MULTUS_CNI_RELEASE} . --push
