FROM keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/golang:1.23-alpine3.19 as builder

RUN apk add --update git make bash gcc musl-dev zip

ARG TERRAFORM_PROVIDER_CCLOUD_VERSION
WORKDIR /go/src/github.com/sapcc/terraform-provider-ccloud
RUN git clone https://github.com/sapcc/terraform-provider-ccloud.git . 
RUN git reset --hard ${TERRAFORM_PROVIDER_CCLOUD_VERSION}
RUN make

ARG TERRAFORM_PROVIDER_OPENSTACK_VERSION
WORKDIR /go/src/github.com/terraform-providers/terraform-provider-openstack
RUN git clone https://github.com/kayrus/terraform-provider-openstack.git . 
RUN git reset --hard ${TERRAFORM_PROVIDER_OPENSTACK_VERSION}
RUN make 

# WORKDIR /go/src/github.com/hashicorp/terraform
# RUN git clone https://github.com/jtopjian/terraform.git --branch backend-swift-auth-update .
# RUN make tools
# RUN make fmt 
# RUN XC_OS=linux XC_ARCH=amd64 make bin

FROM sapcc/kubernikus-terraform:v20181113152146 as terraform11
FROM alpine:3.8
LABEL source_repository="https://github.com/sapcc/kubernikus"

RUN apk add --update make ca-certificates 
COPY --from=builder /go/bin/* /usr/local/bin/
COPY --from=terraform11 /usr/local/bin/terraform /usr/local/bin/

