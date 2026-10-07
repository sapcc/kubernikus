---
title: Local Operator Testing Against a Remote Cluster
---

How to run a local kubernikus operator and API server against an existing control-plane cluster, create a test kluster, and run the e2e suite against it.

## Prerequisites

### Tooling

- `hivemind` — `brew install hivemind`
- `direnv` — `brew install direnv`
- `spore` — for refreshing OpenStack tokens into `_scratch/.curlrc`
- `kubectl` with a kubeconfig context for your control-plane cluster
- Go toolchain (same version as `go.mod`)

### OpenStack project

You need an OpenStack project to create the test kluster in. It must have:

- A router with an external gateway
- A private network and subnet attached to that router
- An SSH keypair or public key to inject into nodes
- Sufficient quota for 3+ compute instances

If the router has more than one private network attached, or the network has more than one subnet, auto-configuration will fail — you must pass `networkID` and `lbSubnetID` explicitly in the kluster spec (see step 3).

### Control-plane cluster

The operator stores kluster CRs and deploys control-plane components (apiserver, wormhole, etcd, …) into a namespace on a Kubernetes cluster. Any existing kubernikus control-plane works: `k-master`, `k-qa-de-1`, etc.

You need a `kubeconfig` context for this cluster. The `.envrc` variable `KS_CONTEXT` must point to it.

### `.envrc` configuration

Copy `.envrc.example` (if present) or set the following variables. `direnv` loads them automatically when you enter the repo:

```bash
export KS_AUTH_URL=https://identity-3.<region>.cloud.sap/v3
export KS_USERNAME=$USER
export KS_USER_DOMAIN_NAME=<your-user-domain>   # e.g. ccadmin or monsoon3
export KS_PROJECT_NAME=<openstack-project>       # project where the operator authenticates
export KS_PROJECT_DOMAIN_NAME=<project-domain>
export KS_NAMESPACE=kubernikus-$USER             # namespace on the control-plane cluster
export KS_CONTEXT=<kubeconfig-context>           # e.g. k-master or k-qa-de-1
export KS_DOMAIN=<kubernikus-domain>             # e.g. kubernikus-master.eu-nl-1.cloud.sap
```

The `.envrc` also refreshes an OpenStack token into `_scratch/.curlrc` via `spore` — adjust the `spore` command to match your region and project.

Store your AD password in the macOS keychain so `kubernikusctl` and the e2e tests can use it:

```bash
security add-generic-password -a $USER -s openstack -p <your-ad-password>
```

After editing `.envrc`:

```bash
direnv allow
```

## 1. Build the binaries

```bash
make bin/darwin/kubernikus bin/darwin/apiserver
```

This stamps the current `git HEAD` as the version into the binary. After any code change on your feature branch, re-run this before restarting hivemind.

## 2. Start the local operator with hivemind

```bash
hivemind
```

This starts two processes defined in `Procfile`:

- `operator` — watches `$KS_NAMESPACE` on `$KS_CONTEXT`, reconciles kluster CRs
- `api` — local kubernikus API server on port 5100

Logs from both are multiplexed in the terminal. `Ctrl-C` stops both. To persist logs:

```bash
hivemind >> /tmp/kubernikus.log
```

## 3. Create a test kluster

The local API server listens on port 5100. Auth uses the OpenStack token in `_scratch/.curlrc`, which `.envrc` keeps fresh via `spore`.

```bash
curl -K _scratch/.curlrc -X POST http://localhost:5100/api/v1/clusters \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "<kluster-name>",
    "spec": {
      "dex": false,
      "backup": "off",
      "sshPublicKey": "ssh-rsa AAAA...",
      "nodePools": [{
        "name": "payload",
        "flavor": "<flavor>",
        "image": "flatcar-stable-amd64",
        "size": 3,
        "availabilityZone": "<az>"
      }],
      "openstack": {
        "routerID": "<router-id>",
        "networkID": "<network-id>",
        "lbSubnetID": "<subnet-id>"
      }
    }
  }'
```

`networkID` and `lbSubnetID` are required when the router or network has multiple attachments. If you omit them and the operator logs `"found 2 networks on router"` or `"found 2 subnets for network"`, delete the kluster and recreate with both fields set.

To find the right IDs, check the operator logs — it lists the networks and subnets it discovered on the router.

To delete a kluster (always use the API, not `kubectl delete kluster` — the API tears down the Helm release and OpenStack resources first):

```bash
curl -K _scratch/.curlrc -X DELETE http://localhost:5100/api/v1/clusters/<kluster-name>
```

## 4. Watch the kluster

The operator deploys control-plane components into `$KS_NAMESPACE` on the control-plane cluster. Watch them come up:

```bash
# all pods in the namespace
kubectl --context $KS_CONTEXT -n $KS_NAMESPACE get pods -w

# wormhole server logs (sidecar in the apiserver pod)
kubectl --context $KS_CONTEXT -n $KS_NAMESPACE logs -f \
  $(kubectl --context $KS_CONTEXT -n $KS_NAMESPACE get pods \
    -l "app=kube-master" \
    -o jsonpath='{.items[0].metadata.name}') \
  -c wormhole
```

The wormhole server starts before nodes join. Once nodes are Ready, the wormhole client daemonset on the nodes connects back to the server and tunnel traffic begins.

### Wormhole image

The operator sets the wormhole container image tag to the current git SHA of the kubernikus repo. This image must exist in the container registry. In CI this is built automatically. For local testing on a feature branch the image likely does not exist yet — patch the deployment to use a known-good image or push one manually:

```bash
# patch to use a pre-built image
kubectl --context $KS_CONTEXT -n $KS_NAMESPACE set image \
  deployment/<kluster-name>-apiserver \
  wormhole=<registry>/<repo>:<tag>

# same for the client daemonset inside the kluster
kubectl --context <kluster-context> -n kube-system set image \
  ds/wormhole wormhole=<registry>/<repo>:<tag>
```

Use a different tag each time you push a new image — `ImagePullPolicy` is `IfNotPresent` so a repeated tag won't pull the new image.

## 5. Get kluster credentials

Once the kluster phase is `Running`:

```bash
bin/darwin/kubernikusctl auth init \
  --url http://localhost:5100 \
  --name <kluster-name>
```

This writes a kubeconfig context for the kluster. Verify:

```bash
kubectl --context <kluster-name> get nodes
```

## 6. Run the e2e suite

The e2e framework authenticates against OpenStack using environment variables. The `.envrc` sets these to the shared e2e tenant — override them with your personal credentials for the project the kluster lives in:

```bash
PW=$(security find-generic-password -a "$USER" -s openstack -w) && \
source .envrc && \
OS_USERNAME=<your-username> \
OS_USER_DOMAIN_NAME=<your-user-domain> \
OS_PROJECT_NAME=<your-project> \
OS_PROJECT_DOMAIN_NAME=<your-project-domain> \
OS_PASSWORD="$PW" \
CP_KUBERNIKUS_URL="" \
go test -v -timeout 55m \
  --kubernikus=http://localhost:5100 \
  --kluster=<kluster-name> \
  --reuse \
  --cleanup=false \
  -run TestRunner/Network \
  ./test/e2e/
```

`CP_KUBERNIKUS_URL=""` skips authentication against the control-plane kubernikus API, which is not needed for local testing.

For the wormhole tunnel test only:

```bash
PW=$(security find-generic-password -a "$USER" -s openstack -w) && \
source .envrc && \
OS_USERNAME=<your-username> \
OS_USER_DOMAIN_NAME=<your-user-domain> \
OS_PROJECT_NAME=<your-project> \
OS_PROJECT_DOMAIN_NAME=<your-project-domain> \
OS_PASSWORD="$PW" \
CP_KUBERNIKUS_URL="" \
go test -v -timeout 15m \
  --kubernikus=http://localhost:5100 \
  --kluster=<kluster-name> \
  --reuse \
  --cleanup=false \
  -run TestRunner/Network/WormholeTunnel \
  ./test/e2e/
```

The `WormholeTunnel` test execs a command into a pod on each node. This goes apiserver→kubelet through the wormhole tunnel — if the tunnel is broken, exec fails.

## 7. Clean up

```bash
curl -K _scratch/.curlrc -X DELETE http://localhost:5100/api/v1/clusters/<kluster-name>
```

Keep the operator running until the kluster disappears from `kubectl --context $KS_CONTEXT -n $KS_NAMESPACE get klusters` — it tears down the Helm release and OpenStack resources before removing the CR.

## Iteration workflow

```bash
# make your change
git add ...
git commit ...

# rebuild
make bin/darwin/kubernikus

# restart hivemind (Ctrl-C, then)
hivemind

# the kluster reconciles within seconds of the operator restarting
# re-run the e2e test
PW=$(security find-generic-password -a "$USER" -s openstack -w) && \
source .envrc && \
OS_USERNAME=<your-username> OS_USER_DOMAIN_NAME=<your-user-domain> \
OS_PROJECT_NAME=<your-project> OS_PROJECT_DOMAIN_NAME=<your-project-domain> \
OS_PASSWORD="$PW" CP_KUBERNIKUS_URL="" \
go test -v -timeout 15m \
  --kubernikus=http://localhost:5100 \
  --kluster=<kluster-name> --reuse --cleanup=false \
  -run TestRunner/Network/WormholeTunnel ./test/e2e/
```
