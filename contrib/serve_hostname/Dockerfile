FROM golang:alpine


WORKDIR /
RUN wget https://raw.githubusercontent.com/kubernetes/kubernetes/416974b8346bb1c219efe871c18a9f29de4fad2d/test/images/serve-hostname/serve_hostname.go
RUN go build -o serve_hostname

FROM alpine:3.8
LABEL source_repository="https://github.com/sapcc/kubernikus"
RUN apk add --no-cache curl bind-tools
EXPOSE 9376
ENTRYPOINT ["/serve_hostname"]
COPY --from=0 /serve_hostname /
