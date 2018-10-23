#!/bin/sh
set -eo pipefail

VIOLATING_FILES=$(goimports -local github.com/sapcc/kubernikus -l $@ | sed /generated/d)
if [ -n "$VIOLATING_FILES" ]; then
  echo "Go code is not formatted in these files:"
  echo "$VIOLATING_FILES"
  echo "Offending lines:"
  goimports -local github.com/sapcc/kubernikus -e -d $VIOLATING_FILES
  exit 1
fi

#Run gofmt to check for possible simplifications (-s flag)
VIOLATING_FILES=$(gofmt -s -l $@)
if [ -n "$VIOLATING_FILES" ]; then
  echo "Go code is not `gofmt -s` formatted in these files:"
  echo "$VIOLATING_FILES"
  echo "Offending lines:"
  gofmt -s -d $VIOLATING_FILES
  exit 1
fi
