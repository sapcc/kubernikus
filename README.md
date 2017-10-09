# Kubernikus

Converged Cloud goes Containers

## Setting up a new Region

### Prepare Tiller with RBAC
```
kubectl -n kube-system create sa tiller
kubectl create clusterrolebinding tiller --clusterrole cluster-admin --serviceaccount=kube-system:tiller
helm init --service-account tiller
```

### Tiller Hacks until Workhole Reverse Tunnel works
`kubectl edit deployment/tiller-deploy`:
```
      volumes:
      - configMap:
          defaultMode: 420
          name: tiller-kubeconfig
        name: kubeconfig
      containers:
      - env:
        - name: TILLER_NAMESPACE
          value: kube-system
        - name: KUBECONFIG
          value: /etc/tiller/kubeconfig
        - name: KUBERNETES_SERVICE_HOST

```

Create Configmap. Edit `server`:
```
cat <<EOF | kubectl create configmap tiller-kubeconfig -n kube-system --from-file -
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    server: https://k-staging-7df1acb1a2f2478eadf3f350d3f44c51.kubernikus-staging.admin.cloud.sap
  name: default
contexts:
- context:
    cluster: default
    namespace: default
    user: default
  name: default
current-context: default
users:
- name: default
  user:
    tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
EOF
```
