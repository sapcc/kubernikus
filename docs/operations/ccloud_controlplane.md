---
title: ControlPlane 
---

## Setting up a new Region

Project is being created using the
[openstack/kubernikus](https://github.com/sapcc/helm-charts/tree/master/openstack/kubernikus)
chart. Install with:

```
helm upgrade kubernikus openstack/kubernikus --namespace monsoon3 --install
```

Complete the project by sharing the external Floating-IP network. Per
convention this is `FloatingIP-external-ccadmin` in the `ccadmin-net-infra`
project in the `ccadmin` domain.

Scope yourself to `cloud_admin` in `ccadmin`:

```
openstack project show -c id -f value --domain ccadmin kubernikus                                                                                                                    
d7df5ce6c37643e49b3c93528c29818b

openstack network show -c id -f value FloatingIP-external-ccadmin                                                                                                                   
c2b999de-adb1-4125-ac3f-f74b9f3a1c63

openstack network rbac create --target-project d7df5ce6c37643e49b3c93528c29818b --action access_as_shared --type network c2b999de-adb1-4125-ac3f-f74b9f3a1c63 
+-------------------+--------------------------------------+
| Field             | Value                                |
+-------------------+--------------------------------------+
| action            | access_as_shared                     |
| id                | 8643f406-6282-46b2-beee-aa6720cf11d5 |
| name              | None                                 |
| object_id         | c2b999de-adb1-4125-ac3f-f74b9f3a1c63 |
| object_type       | network                              |
| project_id        | adc7f04e690a4357a59098c6b2a48db0     |
| target_project_id | d7df5ce6c37643e49b3c93528c29818b     |
+-------------------+--------------------------------------+

```

## Authenticating

Initial kubeconfig setup:
```
kubernikusctl auth init --url https://k-staging.admin.cloud.sap --name k-staging --user-domain-name ccadmin --project-name kubernikus-staging --project-domain-name ccadmin --auth-url https://identity-3.eu-nl-1.cloud.sap --username $USER 
kubernikusctl auth init --url https://k-eu-nl-1.admin.cloud.sap --name k-eu-nl-1 --user-domain-name ccadmin --project-name kubernikus --project-domain-name ccadmin --auth-url https://identity-3.eu-nl-1.cloud.sap --username $USER 
kubernikusctl auth init --url https://k-na-us-1.admin.cloud.sap --name k-na-us-1 --user-domain-name ccadmin --project-name kubernikus --project-domain-name ccadmin --auth-url https://identity-3.na-us-1.cloud.sap --username $USER 
```

## Prepare Tiller with RBAC
```
kubectl -n kube-system create sa tiller
kubectl create clusterrolebinding tiller --clusterrole cluster-admin --serviceaccount=kube-system:tiller
helm init --service-account tiller
```

## Deploy System 

## Deploy Kubernikus
