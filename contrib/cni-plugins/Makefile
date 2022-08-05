CNI_PLUGINS_RELEASE=v1.1.1
IMAGE?=keppel.eu-de-1.cloud.sap/ccloud
ARCH?=amd64


build:
	docker build -t ${IMAGE}/cni-plugins:${CNI_PLUGINS_RELEASE} --build-arg CNI_PLUGINS_RELEASE=${CNI_PLUGINS_RELEASE} .

push:
	docker push ${IMAGE}/cni-plugins:${CNI_PLUGINS_RELEASE}

	

