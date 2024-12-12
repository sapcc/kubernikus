FROM golang:1.22 as builder

WORKDIR /go/src/github.com/kubermatic/k8sniff
ADD . .
RUN CGO_ENABLED=0 go build -ldflags '-s' -o /go/bin/k8sniff .

FROM alpine
LABEL source_repository="https://github.com/sapcc/kubernikus"
COPY --from=builder /go/bin/k8sniff /pipeline/source/k8sniff
CMD ["/pipeline/source/k8sniff"]
