#!/usr/bin/env bash
# Checks that the kube-master apiserver audit policy never writes credentials into the audit
# log: Secret bodies, issued service account tokens, TokenReview requests and similar (see
# SENSITIVE in audit-policy-check.py).
#
# The policy is the audit-policy.yaml key of a ConfigMap in the kube-master chart, rendered
# with test-values.yaml as in test/charts/charts.sh.
#
# Usage: test/charts/audit-policy-conformance.sh
# Requires helm, yq v4 and python3. Exits non-zero if the policy fails or cannot be read.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="$SCRIPT_DIR/../../charts/kube-master"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

helm template -f "$CHART_DIR/test-values.yaml" "$CHART_DIR" \
  | yq eval-all '[select(.kind == "ConfigMap") | select(.data."audit-policy.yaml" != null)]
      | .[0].data."audit-policy.yaml"' - > "$TMP/policy.yaml"
if ! yq -e '.kind == "Policy"' "$TMP/policy.yaml" >/dev/null 2>&1; then
  echo "ERROR: could not extract the audit policy from the rendered kube-master chart"
  exit 1
fi
yq -o=json '.' "$TMP/policy.yaml" > "$TMP/policy.json"

if python3 "$SCRIPT_DIR/audit-policy-check.py" "kube-master/audit-policy.yaml=$TMP/policy.json"; then
  echo "AUDIT POLICY CONFORMANCE: PASSED"
else
  echo "AUDIT POLICY CONFORMANCE: FAILED"
  exit 1
fi
