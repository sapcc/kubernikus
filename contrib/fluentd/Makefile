IMAGE:= keppel.eu-de-1.cloud.sap/ccloud/kubernikus-fluentd
VERSION := v1.14-1

build:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .
push:
	docker push $(IMAGE):$(VERSION)
