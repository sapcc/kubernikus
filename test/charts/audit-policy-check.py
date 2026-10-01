#!/usr/bin/env python3
"""Fail if an audit policy would write credentials into the audit log.

Policies are evaluated the way kube-apiserver evaluates them: the first rule that
matches a request decides its level (k8s.io/apiserver/pkg/audit/policy/checker.go).
For every credential-bearing request in SENSITIVE, each rule that can match it is
checked, up to the first rule that matches it unconditionally. Rules limited to
some users, groups, namespaces or object names count, since they match those.

Usage: audit-policy-check.py LABEL=POLICY.json [LABEL=POLICY.json ...]

Policies are read as JSON (convert with `yq -o=json`). A self-test runs first, so a
broken evaluator fails the check instead of passing every policy.

The same file is used by sapcc/helm-charts (global/cc-gardener/ci/) and
sapcc/kubernikus (test/charts/); keep both copies in sync.
"""

import json
import sys

ALL_VERBS = ("get", "list", "watch", "create", "update", "patch", "delete", "deletecollection")
WRITE_VERBS = ("create", "update", "patch")
LEVELS = ("None", "Metadata", "Request", "RequestResponse")


def stored(level, verb):
    """The object holds the credential: every response and every write request carry it."""
    return level == "RequestResponse" or (level == "Request" and verb in WRITE_VERBS)


def issued(level, verb):
    """The server returns a credential (token, kubeconfig) in the response."""
    return level == "RequestResponse"


def submitted(level, verb):
    """The client sends a credential in the request body."""
    return level in ("Request", "RequestResponse")


# (API group, resource or resource/subresource, verbs, leaks(level, verb))
SENSITIVE = (
    ("", "secrets", ALL_VERBS, stored),
    ("", "serviceaccounts/token", ("create",), issued),
    ("authentication.k8s.io", "tokenreviews", ("create",), submitted),
    ("core.gardener.cloud", "internalsecrets", ALL_VERBS, stored),
    ("core.gardener.cloud", "shoots/adminkubeconfig", ("create",), issued),
    ("core.gardener.cloud", "shoots/viewerkubeconfig", ("create",), issued),
    ("security.gardener.cloud", "workloadidentities/token", ("create",), issued),
    ("metal.ironcore.dev", "bmcsecrets", ALL_VERBS, stored),
)

ALWAYS, SOMETIMES = "always", "sometimes"


def match(rule, group, resource, verb):
    """ALWAYS, SOMETIMES (only some users, namespaces or names) or None.

    Mirrors ruleMatches and ruleMatchesResource in checker.go.
    """
    if rule.get("verbs") and verb not in rule["verbs"]:
        return None
    scope = SOMETIMES if rule.get("users") or rule.get("userGroups") or rule.get("namespaces") else ALWAYS
    if not rule.get("resources"):
        # A nonResourceURLs rule never matches a resource request; any other rule
        # without resources matches every resource (in its namespaces).
        return None if rule.get("nonResourceURLs") and not rule.get("namespaces") else scope
    base, _, sub = resource.partition("/")
    result = None
    for gr in rule["resources"]:
        if gr.get("group", "") not in (group, "*"):
            continue
        names = gr.get("resources") or []
        if not names:
            return scope
        if any(n in ("*", resource, base + "/*") or (sub and n == "*/" + sub) for n in names):
            if not gr.get("resourceNames"):
                return scope
            result = SOMETIMES
    return result


def violations(policy):
    """One line per rule that would log a credential, naming the requests it logs."""
    rules = policy.get("rules") or []
    for i, rule in enumerate(rules):
        if rule.get("level") not in LEVELS:
            raise ValueError(f"rules[{i}] has invalid level {rule.get('level')!r}")
    found = {}
    for group, resource, verbs, leaks in SENSITIVE:
        for verb in verbs:
            for i, rule in enumerate(rules):
                how = match(rule, group, resource, verb)
                if how is None:
                    continue
                if leaks(rule["level"], verb):
                    found.setdefault((i, rule["level"], f"{group or 'core'}/{resource}"), []).append(verb)
                if how == ALWAYS:
                    break
    return [
        f"rules[{i}] level {level} logs {what} bodies on {','.join(verbs)}"
        for (i, level, what), verbs in sorted(found.items())
    ]


def _policy(*rules):
    return {"apiVersion": "audit.k8s.io/v1", "kind": "Policy", "rules": list(rules)}


SECRETS = {"group": "", "resources": ["secrets"]}
GARDENER = "core.gardener.cloud"
# (name, policy, whether it must fail)
SELF_TEST = (
    ("secrets at RequestResponse", _policy({"level": "RequestResponse", "resources": [SECRETS]}), True),
    ("secrets patched at Request", _policy({"level": "Request", "verbs": ["patch"], "resources": [SECRETS]}), True),
    ("catch-all rule", _policy({"level": "None", "nonResourceURLs": ["/healthz*"]}, {"level": "RequestResponse"}), True),
    ("whole core group", _policy({"level": "RequestResponse", "resources": [{"group": ""}]}), True),
    ("any group", _policy({"level": "RequestResponse", "resources": [{"group": "*", "resources": ["secrets"]}]}), True),
    ("resource wildcard on create", _policy({"level": "Request", "verbs": ["create"], "resources": [{"group": "", "resources": ["*"]}]}), True),
    ("catch-all mutations at Request", _policy({"level": "Request", "verbs": ["create", "update", "patch", "delete"]}), True),
    ("secrets/* matches secrets", _policy({"level": "RequestResponse", "resources": [{"group": "", "resources": ["secrets/*"]}]}), True),
    ("user-scoped rule", _policy({"level": "RequestResponse", "users": ["admin"], "resources": [SECRETS]}, {"level": "Metadata"}), True),
    ("name-scoped rule", _policy({"level": "RequestResponse", "resources": [dict(SECRETS, resourceNames=["x"])]}), True),
    ("namespace-scoped catch-all", _policy({"level": "RequestResponse", "namespaces": ["kube-system"]}), True),
    ("service account tokens", _policy({"level": "RequestResponse", "resources": [{"group": "", "resources": ["*/token"]}]}), True),
    ("token reviews at Request", _policy({"level": "Request", "resources": [{"group": "authentication.k8s.io", "resources": ["tokenreviews"]}]}), True),
    ("shoot kubeconfigs", _policy({"level": "RequestResponse", "resources": [{"group": GARDENER, "resources": ["shoots/*"]}]}), True),
    ("earlier Metadata rule wins", _policy({"level": "Metadata", "resources": [SECRETS]}, {"level": "RequestResponse", "resources": [SECRETS]}), False),
    ("secrets read at Request", _policy({"level": "Request", "verbs": ["get", "list", "watch"], "resources": [SECRETS]}, {"level": "Metadata"}), False),
    ("secrets deleted at Request", _policy({"level": "Request", "verbs": ["delete", "deletecollection"], "resources": [SECRETS]}), False),
    ("shoots without subresources", _policy({"level": "RequestResponse", "resources": [{"group": GARDENER, "resources": ["shoots"]}]}), False),
    ("kubeconfig requests at Request", _policy({"level": "Request", "resources": [{"group": GARDENER, "resources": ["shoots/adminkubeconfig"]}]}), False),
    ("non-resource URLs", _policy({"level": "RequestResponse", "nonResourceURLs": ["/*"]}), False),
    ("other API group", _policy({"level": "RequestResponse", "resources": [{"group": "apps"}]}), False),
)


def main(args):
    wrong = [name for name, policy, bad in SELF_TEST if bool(violations(policy)) != bad]
    if wrong:
        print("SELF-TEST FAILED, evaluator is broken: " + "; ".join(wrong))
        return 2
    print(f"self-test passed ({len(SELF_TEST)} cases)")
    failed = False
    for arg in args:
        label, _, path = arg.partition("=")
        try:
            with open(path) as f:
                policy = json.load(f)
            if not isinstance(policy, dict) or policy.get("kind") != "Policy":
                raise ValueError("not an audit Policy")
            found = violations(policy)
        except (OSError, ValueError) as err:
            print(f"ERROR [{label}]: {err}")
            failed = True
            continue
        print(f"{'FAIL' if found else 'ok  '} [{label}]")
        for line in found:
            print(f"       {line}")
        failed = failed or bool(found)
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
