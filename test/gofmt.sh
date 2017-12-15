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
