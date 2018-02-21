---
title: Regional Buildup 
---

## Prepare Values

In the secret repository create values:

  * `admin/values/$REGION.yaml`
  * `kubernikus/$REGION/values/kubernikus.yaml`
  * `kubernikus/$REGION/values/kubernikus-system.yaml`

Create a random password for authentication. Everything else should be a simple
copy/search/replace job.

## Setting up a new Region

Project is being created using the seed chart:
[openstack/kubernikus](https://github.com/sapcc/helm-charts/tree/master/openstack/kubernikus)

Install with:

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

## Admin Control Plane

### Add Pipeline Service User

In `ccadmin/cloud_admin` add the Pipeline service user to the
`ccadmin/kubernikus` project and give it `kubernetes_admin` permissions. While
already here also do the same for the group `CCADMIN_CLOUD_ADMINS`

### Adapt Pipeline

Add authentication blob and new tasks to the `admin` job in the
`ci/pipeline.yaml`. Run the installation of the admin klusters.

### Add DNS Entries

Use `na-us-1/ccadmin/master` to add the following DNS entries:

  * `k-$REGION.admin.cloud.sap.` CNAME	ingress.admin.cloud.sap.	1800
  * `*.k-$REGION.admin.cloud.sap.` CNAME	kubernikus.admin.cloud.sap. 1800

### Rewire Kubernikus Dashboard UI

Scoped as `ccadmin/cloud_admin` create an additional service and endpoint in
the catalog:

```
openstack service create --name kubernikus kubernikus-kubernikus
openstack endpoint create --region $REGION $SERVICEID public https://k-$REGION.admin.cloud.sap
```

### Smoke Test

  1. Check https://k-$REGION.admin.cloud.sap. You should see the Kubernikus splash
page.
  2. Go to `ccadmin/kubernikus` to the Kubernetes tab. It should show you
     a workin UI with no klusters.


## Create Regional Control Plane

Use the UI to create a cluster with the `k-$region` naming scheme in the
`ccadmin/kubernikus` project. Create a `default` pool with 3 nodes in
`m2.xlarge`. Add and select the Kubernikus Master public key.

You should end up with a running kluster and healthy nodes.

### Security Group

Add TCP/UDP Ingress for the source range `198.18.0.0/15`. Required for load
balancers and as a safeguard for DVS agent missed events.

### Authenticating

Done via UI in the Dashboard in the `ccadmin/kubernikus` project

### Prepare Kluster

```
kubectl -n kube-system create sa tiller
kubectl create clusterrolebinding tiller --clusterrole cluster-admin --serviceaccount=kube-system:tiller
helm init --service-account tiller --history-max 5
```

### Adapt Pipeline 

Check the regional `$CONTINENT` jobs. Add tasks for `kubernikus` and
`kubernikus-system` using the authentication blob from earlier. Run the
installation of the continent.

### Load Balancer Config

Once the installation was succesfull, two loadbalancers will appear. Make sure
they have IPs from the `147.*.*.*` range assigned. If not detach, release and
reallocate correct IPs.

Take note that there is one load-balancer with 2 Pools. That is the ingress.
The LB with one pool is the sniffer. Mental mark.

### Add DNS

  * kubernikus-ingress.$REGION.cloud.sap. A	$LB_SNIGGER_IP 1800
  * kubernikus-k8sniff.$REGION.cloud.sap. A $LP_INGRESS_IP 1800	
  * *.kubernikus.$REGION.cloud.sap.	CNAME	kubernikus-k8sniff.$REGION.cloud.sap.	1800	
  * kubernikus.$REGION.cloud.sap. CNAME	kubernikus-ingress.$REGION.cloud.sap.	1800	
  * prometheus.kubernikus.$REGION.cloud.sap.CNAME	kubernikus-ingress.$REGION.cloud.sap.	1800	
  * grafana.kubernikus.$REGION.cloud.sap. CNAME kubernikus-ingress.$REGION.cloud.sap.	1800	

## Relevant URLs

```
for _, region := range []string{"staging", "qa-de-1", "ap-au-1", "eu-de-1", "eu-nl-1", "na-us-1"} {
  ======================================================================
  Admin Control Plane
  ======================================================================
  Kubernikus API: https://k-%v.admin.cloud.sap

  ======================================================================
  Regional Control Plane
  ======================================================================
  Project:        https://dashboard.%v.cloud.sap/ccadmin/kubernikus/home
  Kubernikus API: https://kubernikus.%v.cloud.sap
  Prometheus:     https://prometheus.kubernikus.%v.cloud.sap
  Grafana:        https://grafana.kubernikus.%v.cloud.sap
  Bastion Host:   gateway.kubernikus.%v.cloud.sap
}
```

Poor Man's VPN:
```
sshuttle -r ccloud@gateway.kubernikus.%v.cloud.sap 198.18.0.0/24
```


