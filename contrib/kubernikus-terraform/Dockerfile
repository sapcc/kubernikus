FROM golang:alpine as builder

ARG TERRAFORM_PROVIDER_OPENSTACK_VERSION
ARG TERRAFORM_PROVIDER_CCLOUD_VERSION

RUN apk add --update git make bash gcc musl-dev

WORKDIR /go/src/github.com/sapcc/terraform-provider-ccloud
RUN git clone https://github.com/sapcc/terraform-provider-ccloud.git . 
RUN git reset --hard ${TERRAFORM_PROVIDER_CCLOUD_VERSION}
RUN make

WORKDIR /go/src/github.com/terraform-providers/terraform-provider-openstack
RUN git clone https://github.com/BugRoger/terraform-provider-openstack.git . 
RUN git reset --hard ${TERRAFORM_PROVIDER_OPENSTACK_VERSION}
RUN make 

WORKDIR /go/src/github.com/hashicorp/terraform
RUN git clone https://github.com/jtopjian/terraform.git --branch backend-swift-auth-update .
RUN make tools
RUN make fmt 
RUN XC_OS=linux XC_ARCH=amd64 make bin


FROM alpine:3.8

RUN apk add --update make ca-certificates 
COPY --from=builder /go/bin/* /usr/local/bin/
COPY --from=builder /go/src/github.com/hashicorp/terraform/bin/terraform /usr/local/bin/

