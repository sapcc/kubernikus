#!/bin/sh

echo running kubernikus e2e tests with args: $@
go run test/e2e/*.go $@
