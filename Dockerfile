ARG VERSION=latest

FROM sapcc/kubernikus-binaries:$VERSION as kubernikus-binaries
FROM sapcc/kubernikus-docs:$VERSION as kubernikus-docs

FROM alpine:3.19 as kubernikus
LABEL source_repository="https://github.com/sapcc/kubernikus"
MAINTAINER "Fabian Ruff <fabian.ruff@sap.com>"
RUN apk add --no-cache curl iptables
RUN curl -Lo /bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.5/dumb-init_1.2.5_x86_64 \
	&& chmod +x /bin/dumb-init \
	&& dumb-init -V
COPY etc/*.json /etc/kubernikus/
COPY charts/ /etc/kubernikus/charts
COPY --from=kubernikus-binaries /apiserver /kubernikus /wormhole /usr/local/bin/
#COPY --from=kubernikus-binaries /kubernikusctl /static/binaries/linux/amd64/kubernikusctl
COPY --from=kubernikus-docs /public/docs /static/docs
ENTRYPOINT ["dumb-init", "--"]
CMD ["apiserver"]
