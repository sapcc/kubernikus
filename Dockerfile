FROM alpine:latest
MAINTAINER "Fabian Ruff <fabian.ruff@sap.com>"

RUN apk add --no-cache file
ADD bin/linux/docker.tar /bin/
ENTRYPOINT ["/bin/dumb-init", "--", "/bin/apiserver"]
