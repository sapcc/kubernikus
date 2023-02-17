FROM alpine:3.16
ARG TARGETOS
ARG TARGETARCH
ARG MULTUS_CNI_RELEASE

LABEL source_repository=https://github.com/sapcc/kubernikus/tree/master/contrib/multus-cni
WORKDIR /multus-cni
RUN mkdir -p /multus-cni && \
    wget -O- https://github.com/intel/multus-cni/releases/download/v${MULTUS_CNI_RELEASE}/multus-cni_${MULTUS_CNI_RELEASE}_${TARGETOS}_${TARGETARCH}.tar.gz | tar --strip 1 -xz
