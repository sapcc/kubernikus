---
title: Local Operator Testing Against a Remote Cluster
---

How to run a local operator binary against k-master, create a test kluster, and run the e2e suite against it.

## Prerequisites

- `hivemind` — `brew install hivemind`
- `direnv` — `brew install direnv` and `direnv allow` in the repo root
- Keychain entry `openstack` with your AD password (`security add-generic-password -a $USER -s openstack -p <password>`)
- A valid `kubeconfig` context named `k-master` (the operator reads it via `--context`)

## 1. Build the binaries

```bash
make bin/darwin/kubernikus bin/darwin/apiserver
```

This stamps the current `git HEAD` as the version into the binary. After any code change on your feature branch, re-run this before restarting hivemind.

## 2. Start the local operator with hivemind

The `Procfile` and `.envrc` are already wired up for k-master / `kubernikus-jan` namespace.

```bash
direnv allow        # only needed once, or after .envrc changes
hivemind
```

This starts:
- `operator` — the kubernikus operator watching `kubernikus-jan` namespace on `k-master`
- `api`       — the kubernikus API server (needed if you want to use `kubernikusctl` locally)

Logs from both processes are multiplexed in the terminal. `Ctrl-C` stops both.

## 3. Create a test kluster

The local API server listens on port 5100. Auth is via the OpenStack token in `_scratch/.curlrc` (refreshed automatically by `.envrc`).

```bash
curl -K _scratch/.curlrc -X POST http://localhost:5100/api/v1/clusters \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "jan",
    "spec": {
      "dex": false,
      "backup": "off",
      "sshPublicKey": "ssh-rsa AAAA...",
      "nodePools": [{
        "name": "payload",
        "flavor": "g_c2_m4",
        "image": "flatcar-stable-amd64",
        "size": 3,
        "availabilityZone": "eu-nl-1a"
      }],
      "openstack": {
        "routerID": "2e4beb64-c2b3-4665-bc45-1ea4712c9959"
      }
    }
  }'
```

To delete:

```bash
curl -K _scratch/.curlrc -X DELETE http://localhost:5100/api/v1/clusters/jan
```

## 4. Watch the kluster in k-master

The operator creates a namespace `kubernikus-jan` on k-master. Watch the wormhole server pod specifically:

```bash
# all pods in the namespace
kubectl --context k-master -n kubernikus-jan get pods -w

# wormhole server container logs (inside the apiserver pod)
kubectl --context k-master -n kubernikus-jan logs -f \
  $(kubectl --context k-master -n kubernikus-jan get pods \
    -l "app=kube-master,release=<kluster-name>" \
    -o jsonpath='{.items[0].metadata.name}') \
  -c wormhole

# nftables rules on the control plane node (requires exec into wormhole container)
kubectl --context k-master -n kubernikus-jan exec -it \
  <apiserver-pod> -c wormhole -- nft list ruleset
```

Replace `<kluster-name>` with the name you used in step 3.

## 5. Run the e2e suite against the kluster

The e2e suite requires the kluster to be fully running (nodes Ready) before it can test network connectivity. Use `--reuse` to skip creation and destruction:

```bash
cd test/e2e && \
  go test -v -timeout 55m \
    --kubernikus=$KUBERNIKUS_URL \
    --kluster=<kluster-name> \
    --reuse \
    --cleanup=false \
    -run TestRunner/Network
```

For the wormhole-specific test only:

```bash
cd test/e2e && \
  go test -v -timeout 15m \
    --kubernikus=$KUBERNIKUS_URL \
    --kluster=<kluster-name> \
    --reuse \
    --cleanup=false \
    -run TestRunner/Network/WormholeTunnel
```

The `WormholeTunnel` test polls `RouteBroken=False` with a heartbeat no older than 3 minutes on every node. It times out after 5 minutes — give the kluster ~2 minutes after nodes are Ready before running it.

Required environment variables (already set by `.envrc`):

| Variable | Purpose |
|----------|---------|
| `OS_AUTH_URL` | Identity endpoint for the e2e test tenant |
| `OS_USERNAME` | e2e test user |
| `OS_PASSWORD` | e2e test password |
| `OS_USER_DOMAIN_NAME` | Domain of the test user |
| `OS_PROJECT_NAME` | Project the kluster lives in |
| `OS_PROJECT_DOMAIN_NAME` | Domain of the project |
| `CP_KUBERNIKUS_URL` | Kubernikus API for the control plane cluster (k-master) |
| `CP_KLUSTER` | Name of the control plane kluster (`k-master`) |

## 6. Clean up

```bash
curl -K _scratch/.curlrc -X DELETE http://localhost:5100/api/v1/clusters/<kluster-name>
```

Keep the operator running until the kluster disappears — it tears down the Helm release and OpenStack resources before removing the CR.

## Iteration workflow on a feature branch

```bash
# make your change
git add ...
git commit ...

# rebuild
make bin/darwin/kubernikus

# restart only the operator (hivemind: Ctrl-C on the operator line, or restart all)
hivemind

# kluster will reconcile within seconds of the operator restarting
# re-run the specific e2e test
cd test/e2e && go test -v -timeout 15m --kubernikus=$KUBERNIKUS_URL \
  --kluster=<kluster-name> --reuse --cleanup=false \
  -run TestRunner/Network/WormholeTunnel
```
