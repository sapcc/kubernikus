ARG VERSION=latest
FROM sapcc/kubernikus-binaries:$VERSION as kubernikus-binaries

FROM alpine:3.8
COPY --from=kubernikus-binaries /kubernikusctl /usr/local/bin/
CMD ["kubernikusctl"]
