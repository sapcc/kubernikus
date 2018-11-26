IMAGE:= sapcc/serve_hostname
VERSION := 1.2-alpine

build:
	docker build -t $(IMAGE):$(VERSION) .
push:
	docker push $(IMAGE):$(VERSION)
