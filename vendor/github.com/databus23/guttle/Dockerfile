FROM golang:1.10.3-alpine3.8 as builder
WORKDIR /go/src/github.com/databus23/guttle/
RUN apk add --no-cache make
COPY . .
#ARG VERSION
RUN make all

FROM alpine:3.6
MAINTAINER "Fabian Ruff <fabian.ruff@sap.com>"
RUN apk add --no-cache curl iptables
COPY --from=builder /go/src/github.com/databus23/guttle/bin/linux/ /usr/local/bin/
CMD ["guttle", "server"]
