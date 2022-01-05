FROM keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/alpine:3.13
LABEL source_repository="https://github.com/sapcc/kubernikus"

ARG HUGO_VERSION=0.30.2

RUN apk add --no-cache curl
RUN curl -Lo hugo.tar.gz https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_${HUGO_VERSION}_Linux-64bit.tar.gz 
RUN tar xvf hugo.tar.gz hugo -C /usr/local/bin
RUN curl -Lo /usr/local/bin/yaml2json https://github.com/bronze1man/yaml2json/releases/download/v1.2/yaml2json_linux_amd64 && \
  chmod +x /usr/local/bin/yaml2json
COPY data /

ONBUILD COPY docs /content
ONBUILD RUN hugo --baseURL "/docs"       --destination /public/docs
ONBUILD RUN hugo --baseURL "/kubernikus" --destination /public/kubernikus

ONBUILD COPY swagger.yml /
ONBUILD RUN yaml2json < /swagger.yml > /public/docs/api/swagger.json
ONBUILD RUN yaml2json < /swagger.yml > /public/kubernikus/api/swagger.json
