ARG VERSION=latest
FROM sapcc/kubernikus-binaries:$VERSION as kubernikus-binaries

FROM alpine:3.8

ARG KUBERNETES_VERSION=v1.10.9
ARG HELM_VERSION=v2.11.0

RUN apk add --no-cache --virtual=build-dependencies bash jq curl ca-certificates make

RUN curl -fLo /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl \
    && chmod +x /usr/bin/kubectl /usr/bin/kubectl \
    && /usr/bin/kubectl version -c

RUN curl http://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar -xz \
      && mv linux-amd64/helm /usr/bin/ \
      && rm -rf linux-amd64 \
      && helm version -c \
      && helm init -c \
      && curl -L https://github.com/databus23/helm-diff/releases/download/v2.11.0%2B2/helm-diff-linux.tgz | tar -xz -C ~/.helm/plugins

ADD kubectl helm /usr/local/bin/
COPY --from=kubernikus-binaries /kubernikusctl /usr/local/bin/
