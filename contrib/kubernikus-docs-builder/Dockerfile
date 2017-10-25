FROM alpine:3.6

ARG HUGO_VERSION=0.30.2

RUN apk add --no-cache curl
RUN curl -Lo hugo.tar.gz https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_${HUGO_VERSION}_Linux-64bit.tar.gz 
RUN tar xvf hugo.tar.gz hugo -C /usr/local/bin
COPY data /

ONBUILD COPY docs /content
ONBUILD RUN hugo --debug

