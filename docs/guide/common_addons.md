---
title: Common Addons
---

## Kubernetes Dashboard

[Kubernetes Dashboard](https://github.com/kubernetes/dashboard) is a general
purpose, web-based UI for Kubernetes clusters. It allows users to manage
applications running in the cluster and troubleshoot them, as well as manage
the cluster itself.


### Installation

[Installation](https://github.com/kubernetes/dashboard) is straight forward:

```
kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/master/src/deploy/recommended/kubernetes-dashboard.yaml
```

### Granting Permissions

You can grant full admin privileges to Dashboard's Service Account by creating
below `ClusterRoleBinding`.

```
kubectl create clusterrolebinding kubernetes-dashboard --clusterrole=cluster-admin --serviceaccount=kube-system:kubernetes-dashboard
```

### Accessing the Dashboard

To access Dashboard from your local workstation you must create a secure
channel to your Kubernetes cluster. Run the following command:

```
kubectl proxy
```

Now access Dashboard at:

[http://localhost:8001/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/.](http://localhost:8001/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/.)

### Exposig the Dashboard

In order to expose the Dashboard without the local proxy, we need to:

  * Create a service of type `LoadBalancer`
  * Open the security groups for load-balancer to node communication
  * Assign a floating IP

Let's create the service:

```
kubectl expose deployment kubernetes-dashboard --namespace kube-system --type=LoadBalancer --name kubernete-dashboard-external --port=443
```

This will create a Kubernetes service that exposes the dashboard on
a high-level service port on each node of the cluster. Additionally,
a load-balancer is created in OpenStack which points to each node.

![Load Balancer](https://raw.githubusercontent.com/sapcc/kubernikus/master/assets/images/docs/containers/kubernetes/loadbalancer0.png)

As the load-balancers are not in the default instance group, they need to be
added to the security group explicitly. Without this the health monitors won't
be able to reach the node port and the listener will be offline.

![Security Group](https://raw.githubusercontent.com/sapcc/kubernikus/master/assets/images/docs/containers/kubernetes/loadbalancer1.png)

Once the health monitors turn healthy, attaching a floating IP will make the
dashboard accessible from the outside via `https` on port `443`.

![FIP](https://raw.githubusercontent.com/sapcc/kubernikus/master/assets/images/docs/containers/kubernetes/loadbalancer2.png)

~> This setup exposes a unauthenticated Dashboard with full access to the cluster. This is a security risk. See the [Access Control](https://github.com/kubernetes/dashboard/wiki/Access-control) documentation for more info.
