FROM alpine:3.9

ARG TERRAFORM_VERSION
ARG TERRAFORM_PROVIDER_OPENSTACK_VERSION
ARG TERRAFORM_PROVIDER_CCLOUD_VERSION

RUN apk add --update make ca-certificates 
ADD https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip /usr/local/bin/
ADD https://github.com/sapcc/terraform-provider-ccloud/releases/download/v${TERRAFORM_PROVIDER_CCLOUD_VERSION}/terraform-provider-ccloud_linux_amd64.zip /usr/local/bin/
ADD https://releases.hashicorp.com/terraform-provider-openstack/${TERRAFORM_PROVIDER_OPENSTACK_VERSION}/terraform-provider-openstack_${TERRAFORM_PROVIDER_OPENSTACK_VERSION}_linux_amd64.zip /usr/local/bin/
RUN for i in /usr/local/bin/*.zip; do unzip -qd /usr/local/bin/ $i; done && rm /usr/local/bin/*.zip
