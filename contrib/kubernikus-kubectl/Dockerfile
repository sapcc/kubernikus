ARG VERSION=latest
FROM sapcc/kubernikus-binaries:$VERSION as kubernikus-binaries

FROM alpine:3.6

ARG KUBERNETES_VERSION
ARG HELM_VERSION

RUN apk add --no-cache --virtual=build-dependencies bash jq curl ca-certificates 

RUN curl -fLo /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl \
    && chmod +x /usr/bin/kubectl /usr/bin/kubectl \
    && /usr/bin/kubectl version -c

RUN curl http://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar -xz \
      && mv linux-amd64/helm /usr/bin/ \
      && rm -rf linux-amd64 \
      && helm version -c \
      && helm init -c

ADD kubectl helm /usr/local/bin/
COPY --from=kubernikus-binaries /kubernikusctl /usr/local/bin/
