IMAGE:= sapcc/k8sniff
VERSION := e7435d989925e8559b0e5ca26da69f84a1035c32

build:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .
push:
	docker push $(IMAGE):$(VERSION)
