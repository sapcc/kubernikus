#!/bin/sh
set -eo pipefail

VIOLATING_FILES=$(goimports -local github.com/sapcc/kubernikus -l $@ | sed /generated/d)
if [ -n "$VIOLATING_FILES" ]; then
  echo "Go code is not formatted:"
  goimports -e -d $@
  exit 1
fi
