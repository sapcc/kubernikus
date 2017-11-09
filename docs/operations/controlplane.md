---
title: ControlPlane 
---

## Setting up a new Region

  * Create a new Openstack project, e.g. `ccadmin/kubernikus`
  * Create service user: `openstack user create kubernikus --domain Default t--password abc123`
  * Create a new network: `openstack network create kubernikus`
  * Create a subnet: `openstack subnet create --network kubernikus --subnet-range 198.18.0.0/24 kubernikus`
  * Create Router
  * Assign administrative roles: `openstack role add --user kubernikus --user-domain Default --domain ccadmin admin`

```
openstack role add --user kubernikus --user-domain Default --project cloud_admin --project-domain ccadmin admin
openstack role add --user kubernikus --user-domain Default --project cloud_admin --project-domain ccadmin cloud_network_admin
openstack role add --user kubernikus --user-domain Default --project cloud_admin --project-domain ccadmin cloud_compute_admin
openstack role add --user kubernikus --user-domain Default --project cloud_admin --project-domain ccadmin cloud_dns_admin
```


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
