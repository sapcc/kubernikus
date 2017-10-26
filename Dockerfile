FROM golang:1.9.0-alpine3.6 as builder
WORKDIR /go/src/github.com/sapcc/kubernikus/
RUN apk add --no-cache make
COPY . .
ARG VERSION
RUN make all

FROM scratch as kubernikus-binaries
COPY --from=builder /go/src/github.com/sapcc/kubernikus/bin/linux/* /

FROM alpine:3.6 as kubernikusctl
COPY --from=kubernikus-binaries /kubernikusctl /usr/local/bin/
CMD ["kubernikusctl"]

FROM sapcc/kubernikus-docs-builder as kubernikus-docs

FROM alpine:3.6 as kubernikus
MAINTAINER "Fabian Ruff <fabian.ruff@sap.com>"
RUN apk add --no-cache curl iptables
RUN curl -Lo /bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.0/dumb-init_1.2.0_amd64 \
	&& chmod +x /bin/dumb-init \
	&& dumb-init -V
COPY charts/ /etc/kubernikus/charts
COPY --from=kubernikus-binaries /apiserver /kubernikus /wormhole /usr/local/bin/
COPY --from=kubernikus-binaries /kubernikusctl /static/binaries/linux/amd64/kubernikusctl
COPY --from=kubernikus-docs /public /static/docs
ENTRYPOINT ["dumb-init", "--"]
CMD ["apiserver"]
