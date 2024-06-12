IMAGE:=keppel.eu-de-1.cloud.sap/ccloud/k8sniff
VERSION:=$(shell git rev-parse --verify HEAD)

build:
	docker build --network=host -t $(IMAGE):$(VERSION) .

push:
	docker push $(IMAGE):$(VERSION)
