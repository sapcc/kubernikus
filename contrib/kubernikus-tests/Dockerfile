FROM keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/golang:1.25-alpine3.21
LABEL source_repository="https://github.com/sapcc/kubernikus"

WORKDIR /go/src/github.com/sapcc/kubernikus/

RUN apk add --no-cache make git curl bash

RUN curl -Lf https://get.helm.sh/helm-v3.8.2-linux-amd64.tar.gz\
	| tar --strip-components=1 -C /usr/local/bin -zxv \
	&& helm version -c

RUN curl -Lf https://github.com/prometheus/prometheus/releases/download/v2.12.0/prometheus-2.12.0.linux-amd64.tar.gz \
	| tar --strip-components=1 -C /usr/local/bin -zxv prometheus-2.12.0.linux-amd64/promtool \
	&& promtool --version

RUN curl -Lfo /usr/local/bin/yq https://github.com/mikefarah/yq/releases/download/2.2.1/yq_linux_amd64 \
	&& chmod +x /usr/local/bin/yq

RUN curl -Lf https://github.com/kyoh86/exportloopref/releases/download/v0.1.8/exportloopref_0.1.8_linux_amd64.tar.gz \
	| tar -C /usr/local/bin -zxv exportloopref \
	&& chmod +x /usr/local/bin/exportloopref
