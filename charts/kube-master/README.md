kube-master chart
=================
This chart installs the master components of a kubernetes cluster:

 * apiserver
 * controller-manager
 * scheduler
 * etcd

Configuration
=============

| Parameters                      | Description                                             | Default                           |
| ------------------------------- | ------------------------------------------------------- | --------------------------------- |
| `image.repository`              | `hyperkube` image repository                            | quay.io/coreos/hyperkube          |
| `image.tag`                     | `hyperkube` image tag                                   | v1.6.1_coreos.0                   |
| `openstack.{}`                  | OpenStack Cloudprovider stuff                           | *See values.yaml for details*     |
| `certsSecretName`               | name of the secret holding the certificates (see below) | secrets *(managed by helm)*       | 
| `certs`                         | certificates/keys managed by helm (see below)           | {}                                |
| `api.ingressHost`               | Hostname for the apiserver ingress resources            | - *(ingress disabled by default)* |
| `api.flags`                     | flags for the apiserver                                 | *See values.yaml*                 |
| `api.resources`                 | apiserver CPU/Memory resource requests/limits           | Memory: `256Mi`, CPU: `100m`      |
| `controllerManager.flags`       | flags for the controllerManager                         | *See values.yaml*                 |
| `controllerManager.resource`    | controller-manager CPU/Memory resource requests/limits  | Memory: `256Mi`, CPU: `100m`      |
| `scheduler.flags`               | flags for the scheduler                                 | *See values.yaml*                 |
| `scheduler.resources`           | scheduler CPU/Memory resource requests/limits           | Memory: `256Mi`, CPU: `100m`      |
| `etcd.image.repository`         | `etcd` image repository                                 | gcr.io/google_containers/etcd     |
| `etcd.image.tag`                | `etcd` image tag                                        | 3.0.17                            |
| `etcd.persistence.enabled`      | Use a PVC to persist etcd data                          | true                              |
| `etcd.persistence.accessMode`   | PVVs access mode                                        | ReadWriteMany                     |
| `etcd.persistence.size`         | Size of the volume                                      | 10Gi                              |
| `etcd.persistence.exitingClaim` | Re-use an exiting claim (not managed by helm)           | -                                 |
| `etcd.resources`                | etcd CPU/Memory resource requests/limits                | Memory: `256Mi`, CPU: `100m`      |

### Certificates
You have to specify a lot of certificates. You can either do that via helm values or specify an exiting secret containing all necessary keys and certificates.

Provide the following certificates either in the `certs` value subsection or in a secret specified via `certsSecretName`.

* `kube-clients-ca.pem`
* `etcd-clients-ca.pem`
* `etcd-clients-apiserver.pem`
* `etcd-clients-apiserver.key`
* `kube-clients-ca.pem`
* `kube-clients-ca.key`
* `kube-controller-manager.pem`
* `kube-controller-manager.key`
* `kube-scheduler.pem`
* `kube-scheudler.key`
* `kube-nodes-ca.pem`
* `kube-nodes-ca.key`
* `kube-nodes-apiserver.pem`
* `kube-nodes-apiserver.key`
* `tls-ca.pem`
* `tls-kube-apiserver.pem`
* `tls-kube-apiserver.key`
* `tls-sni.pem`
* `tls-sni.key`