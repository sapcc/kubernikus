GOOS     ?= $(shell go env | grep GOOS | cut -d'"' -f2)
BINARIES := guttle

SRCDIRS  := . cmd
PACKAGES := $(shell find $(SRCDIRS) -type d)
GOFILES  := $(addsuffix /*.go,$(PACKAGES))
GOFILES  := $(wildcard $(GOFILES))

all: $(BINARIES:%=bin/$(GOOS)/%)

bin/%: $(GOFILES) Makefile
	GOOS=$(*D) GOARCH=amd64 go build $(GOFLAGS) -v -i -o $(@D)/$(@F) ./cmd

image:
	docker build -t guttle .

test: image
	docker run --rm -it --name guttle --cap-add NET_ADMIN -p 9090:9090 -v $(CURDIR)/scripts/:/scripts guttle /scripts/test.sh

test-client: bin/$(GOOS)/guttle
	bin/$(GOOS)/guttle client --server localhost:9090 --listen-addr localhost:8080

