IMAGE:= sapcc/serve-hostname-amd64
VERSION := 1.2-alpine

build:
	docker build -t $(IMAGE):$(VERSION) .
push:
	docker push $(IMAGE):$(VERSION)
