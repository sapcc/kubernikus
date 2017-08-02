FROM golang:1.8.3-alpine3.6 as builder
WORKDIR /go/src/github.com/sapcc/kubernikus/
COPY . .
RUN apk add --no-cache make
ARG VERSION
RUN make all

FROM alpine:3.6
MAINTAINER "Fabian Ruff <fabian.ruff@sap.com>"
RUN apk add --no-cache curl
RUN curl -Lo /usr/bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.0/dumb-init_1.2.0_amd64 \
	&& chmod +x /usr/bin/dumb-init \
	&& dumb-init -V
COPY --from=builder /go/src/github.com/sapcc/kubernikus/bin/linux/ /usr/local/bin/
COPY charts/ /etc/kubernikus/charts
ENTRYPOINT ["/bin/dumb-init", "--"]
CMD ["/bin/apiserver"]
