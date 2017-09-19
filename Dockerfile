FROM golang:1.9.0-alpine3.6 as builder
WORKDIR /go/src/github.com/sapcc/kubernikus/
RUN apk add --no-cache make
COPY . .
ARG VERSION
RUN make all

FROM alpine:3.6
MAINTAINER "Fabian Ruff <fabian.ruff@sap.com>"
RUN apk add --no-cache curl iptables
RUN curl -Lo /bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.0/dumb-init_1.2.0_amd64 \
	&& chmod +x /bin/dumb-init \
	&& dumb-init -V
COPY charts/ /etc/kubernikus/charts
COPY --from=builder /go/src/github.com/sapcc/kubernikus/bin/linux/ /usr/local/bin/
ENTRYPOINT ["dumb-init", "--"]
CMD ["apiserver"]
