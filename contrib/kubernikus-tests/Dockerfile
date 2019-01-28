FROM golang:1.11.5-alpine3.8

WORKDIR /go/src/github.com/sapcc/kubernikus/

RUN apk add --no-cache make git curl bash

RUN curl -Lf https://storage.googleapis.com/kubernetes-helm/helm-v2.10.0-linux-amd64.tar.gz \
		| tar --strip-components=1 -C /usr/local/bin -zxv \
		&& helm version -c

RUN curl -Lf https://github.com/prometheus/prometheus/releases/download/v2.4.2/prometheus-2.4.2.linux-amd64.tar.gz \
		| tar --strip-components=1 -C /usr/local/bin -zxv prometheus-2.4.2.linux-amd64/promtool \
		&& promtool --version

RUN curl -Lfo /usr/local/bin/yq https://github.com/mikefarah/yq/releases/download/2.2.1/yq_linux_amd64 \
		&& chmod +x /usr/local/bin/yq
