FROM alpine as builder

WORKDIR /tmp

ARG KUBERNETES_VERSION
ARG CNI_PLUGINS_RELEASE
ARG MULTUS_RELEASE
ARG ARCH

RUN apk add --no-cache curl

RUN wget https://storage.googleapis.com/kubernetes-release/release/$KUBERNETES_VERSION/bin/linux/amd64/kubelet && \
    chmod +x kubelet && \
    wget https://storage.googleapis.com/kubernetes-release/release/$KUBERNETES_VERSION/bin/linux/amd64/kubectl && \
    chmod +x kubectl

RUN	mkdir -p cni-bin/bin && \
    curl -sSL --retry 5 https://github.com/containernetworking/plugins/releases/download/$CNI_PLUGINS_RELEASE/cni-plugins-linux-$ARCH-$CNI_PLUGINS_RELEASE.tgz | tar -xz -C cni-bin/bin

# todo fix multus release env var with undescore
RUN curl -sSL --retry 5 https://github.com/intel/multus-cni/releases/download/v$MULTUS_RELEASE/multus-cni_3.8_linux_amd64.tar.gz | tar --strip 1 -xz -C cni-bin/bin

FROM k8s.gcr.io/build-image/debian-hyperkube-base-amd64:v1.1.0 

LABEL source_repository="https://github.com/sapcc/kubernikus"

RUN rm -rf /usr/local/bin/* /apiserver /cloud-controller-manager /controller-manager /kubectl /kubelet /proxy /scheduler

RUN apt-get update && apt-get install -y \
    iproute2 \
    net-tools \
    dnsutils \
    tcpdump \
    iputils-ping \
    traceroute \
    bridge-utils \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /tmp/kubelet /usr/local/bin/kubelet
COPY --from=builder /tmp/kubectl /usr/local/bin/kubectl
COPY --from=builder /tmp/cni-bin/bin/ /opt/cni/bin
