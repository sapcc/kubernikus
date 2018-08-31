FROM golang:alpine as builder

ARG TERRAFORM_PROVIDER_OPENSTACK_VERSION
ARG TERRAFORM_PROVIDER_CCLOUD_VERSION

RUN apk add --update git make bash

WORKDIR /go/src/github.com/sapcc/terraform-provider-ccloud
RUN git clone https://github.com/sapcc/terraform-provider-ccloud.git . 
RUN git reset --hard ${TERRAFORM_PROVIDER_CCLOUD_VERSION}
RUN make

WORKDIR /go/src/github.com/terraform-providers/terraform-provider-openstack
RUN git clone https://github.com/BugRoger/terraform-provider-openstack.git . 
RUN git reset --hard ${TERRAFORM_PROVIDER_OPENSTACK_VERSION}
RUN make 

FROM alpine:3.8

ARG TERRAFORM_VERSION

RUN apk add --update make terraform=${TERRAFORM_VERSION}
COPY --from=builder /go/bin/* /usr/local/bin/
