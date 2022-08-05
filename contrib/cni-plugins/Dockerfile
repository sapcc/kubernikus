FROM alpine
ARG TARGETARCH
ARG CNI_PLUGINS_RELEASE

LABEL source_repository=https://github.com/sapcc/kubernikus/tree/master/contrib/cni-plugins
WORKDIR /cni-plugins
RUN mkdir -p /cni-plugins && \
    wget -O- https://github.com/containernetworking/plugins/releases/download/${CNI_PLUGINS_RELEASE}/cni-plugins-linux-${TARGETARCH}-${CNI_PLUGINS_RELEASE}.tgz | tar -xz
