FROM golang:1.12-alpine

WORKDIR /go/src/github.com/kubermatic/k8sniff

RUN apk add --no-cache curl git \
    && curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
ARG VERSION=master
RUN git clone https://github.com/kubermatic/k8sniff.git . \
    && git checkout $VERSION

RUN dep ensure

RUN go build -v -o k8sniff .

FROM alpine:3.9

RUN apk add --no-cache ca-certificates
COPY --from=0 /go/src/github.com/kubermatic/k8sniff /pipeline/source/k8sniff
