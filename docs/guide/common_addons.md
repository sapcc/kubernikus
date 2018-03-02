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

Skip the selection of Kubeconfig or Token:

![Selection](https://raw.githubusercontent.com/sapcc/kubernikus/master/assets/images/docs/containers/kubernetes/selection.png)

### Exposing the Dashboard

In order to expose the Dashboard without the local proxy, we need to:

  * Create a service of type `LoadBalancer`
  * Open the security groups for load-balancer to node communication
  * Assign a floating IP

Let's create the service:

```
kubectl expose deployment kubernetes-dashboard --namespace kube-system --type=LoadBalancer --name kubernete-dashboard-external --port=443 --target-port=8443
```

This will create a Kubernetes service that exposes the dashboard on
a high-level service port on each node of the cluster. Additionally,
a load-balancer is created in OpenStack which points to each node.

![Load Balancer](https://raw.githubusercontent.com/sapcc/kubernikus/master/assets/images/docs/containers/kubernetes/loadbalancer0.png)

As the load-balancers are not in the default instance group, they need to be
added to the security group explicitly. Without this, the health monitors won't
be able to reach the node port and the listener will be offline.

![Security Group](https://raw.githubusercontent.com/sapcc/kubernikus/master/assets/images/docs/containers/kubernetes/loadbalancer1.png)

Once the health monitors turn healthy, attaching a floating IP will make the
dashboard accessible from the outside via `https` on port `443`.

![FIP](https://raw.githubusercontent.com/sapcc/kubernikus/master/assets/images/docs/containers/kubernetes/loadbalancer2.png)

~> This setup exposes an unauthenticated Dashboard with full access to the cluster. This is a security risk. See the [Access Control](https://github.com/kubernetes/dashboard/wiki/Access-control) documentation for more info.

## Private Docker Registry in Kubernikus
You can create a private docker registry in your Kubernikus cluster to store your Docker images. 

### How it works
The private registry runs as a Pod in your cluster. A proxy on each node is exposing a port onto the node (via a hostPort), which Docker accepts as "secure", since it is accessed by localhost.

### Create a persisten volume claim
There is already a default storageClass and your cluster knows that storage exists. You just have to create a persistent volume claim to claim the storage. 

```
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: registry-storage
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

### Run the registry
Now you can run the Docker registry:

```
apiVersion: v1
kind: ReplicationController
metadata:
  name: kube-registry-v0
  labels:
    k8s-app: kube-registry-upstream
    version: v0
    kubernetes.io/cluster-service: "true"
spec:
  replicas: 1
  selector:
    k8s-app: kube-registry-upstream
    version: v0
  template:
    metadata:
      labels:
        k8s-app: kube-registry-upstream
        version: v0
        kubernetes.io/cluster-service: "true"
    spec:
      containers:
      - name: registry
        image: registry:2.5.1
        resources:
          limits:
            cpu: 100m
            memory: 100Mi
          requests:
            cpu: 100m
            memory: 100Mi
        env:
        - name: REGISTRY_HTTP_ADDR
          value: :5000
        - name: REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY
          value: /var/lib/registry
        volumeMounts:
        - name: image-store
          mountPath: /var/lib/registry
        ports:
        - containerPort: 5000
          name: registry
          protocol: TCP
      volumes:
      - name: image-store
        persistentVolumeClaim: 
          claimName: registry-storage
```

### Expose registry in the cluster
```
apiVersion: v1
kind: Service
metadata:
  name: kube-registry
  labels:
    k8s-app: kube-registry-upstream
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: "KubeRegistry"
spec:
  selector:
    k8s-app: kube-registry-upstream
  ports:
  - name: registry
    port: 5000
    protocol: TCP
```

### Expose the registry on each node
Now that there is a running Service, you need to expose it onto each Kubernetes Node so that Docker will see it as localhost.

```
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: registry-proxy-v0
  labels:
    k8s-app: kube-registry-proxy
    kubernetes.io/cluster-service: "true"
    version: v0.4
spec:
  template:
    metadata:
      labels:
        k8s-app: kube-registry-proxy
        kubernetes.io/name: "kube-registry-proxy"
        kubernetes.io/cluster-service: "true"
        version: v0.4
    spec:
      containers:
      - name: kube-registry-proxy
        image: gcr.io/google_containers/kube-registry-proxy:0.4
        resources:
          limits:
            cpu: 100m
            memory: 50Mi
        env:
        - name: REGISTRY_HOST
          value: kube-registry
        - name: REGISTRY_PORT
          value: "5000"
        ports:
        - name: registry
          containerPort: 80
          hostPort: 5000
```

### Access registry from outside
Through a ssh tunnel you can push or pull images from your cluster registry. At first, export your local ip: 
```
export local_ip=$(ipconfig getifaddr en0)
```
Add `${local_ip}:5000` to your local docker daemon config insecure registries. Save and restart docker daemon.

After start the ssh tunnel:
```
ssh -N -L '*:5000:localhost:5000' <username>@<remote-registry-server>
```
Now you can pull or push images
```
docker pull ${local_ip}:5000/<user>/<image>
```



