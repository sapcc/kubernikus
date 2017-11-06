---
title: Logging 
---

## Setting up a new Region

### Prepare Tiller with RBAC
```
kubectl -n kube-system create sa tiller
kubectl create clusterrolebinding tiller --clusterrole cluster-admin --serviceaccount=kube-system:tiller
helm init --service-account tiller
```

## Authenticating

Initial kubeconfig setup:
```
kubernikusctl auth init --url https://k-staging.admin.cloud.sap --name k-staging --user-domain-name ccadmin --project-name kubernikus-staging --project-domain-name ccadmin --auth-url https://identity-3.eu-nl-1.cloud.sap --username <USER> 
kubernikusctl auth init --url https://k-eu-nl-1.admin.cloud.sap --name k-eu-nl-1 --user-domain-name ccadmin --project-name kubernikus --project-domain-name ccadmin --auth-url https://identity-3.eu-nl-1.cloud.sap --username <USER> 
```

Refresh certificates with:
```
kubernikusctl auth refresh
```

Automate in Fish with:
```
function _kubectl
  test -n "$KUBECTL_CONTEXT"; or set -x KUBECTL_CONTEXT (kubectl config current-context)

  if count $argv > /dev/null
    kubernikusctl auth refresh --context $KUBECTL_CONTEXT
  end

  if test -n "$KUBECTL_NAMESPACE"
    eval kubectl --context $KUBECTL_CONTEXT --namespace $KUBECTL_NAMESPACE $argv
  else
    eval kubectl --context $KUBECTL_CONTEXT $argv
  end
end
```
