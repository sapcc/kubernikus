---
title: Authentication
---

## Intro

In Kubernetes regular end-users are assumed to be managed by an outside,
independent service. In this regard, Kubernetes does not have objects which
represent normal user accounts. Regular users cannot be added to a cluster
through an API call.

API requests are tied to either a normal user or a service account, or are
treated as anonymous requests. This means every process inside or outside the
cluster, from a human user typing kubectl on a workstation, to kubelets on
nodes, to members of the control plane, must authenticate when making requests
to the API server, or be treated as an anonymous user.

### User-Management

For Kubernikus clusters the user management is handled by OpenStack's Identity
Service (Keystone). Only users that have been given a `os:kubernikus_admin` or
`os:kubernikus_member` roles by an Keystone administrator are allowed to interact
with the service or clusters.

### Authentication

The authentication against Kubernikus clusters is based on x509 certificates.
Encoded into the certificate's `Common Name` field is the user name. The
certificate's `organization` fields indicate the user's OpenStack role
assignments.

This effectively maps OpenStack roles to Kubernetes groups.

These certificates are generated. They can be retrieved via UI or API. In order
to allow for revocation of authorization the certificates are short lived. They
automatically expire after 24h. Therefore they need to be periodically
refreshed.

### Authorizations

Using the `user` and `groups` provided by the authentication mechanism it is
then possible to use [Kubernetes
RBAC](https://kubernetes.io/docs/admin/authorization/rbac/) to define
authorizations within Kubernetes.

By distinguishing between users as well as `kubernikus_admin` and `kubernikus_member`
roles/groups it is possible to assign different Kubernetes roles to groups or
individual users.

## Manage Roles in OpenStack

Users with `Keystone Administrator` role are allowed to change user role
assignments in a project.

![User Role
Assignments](https://raw.githubusercontent.com/sapcc/kubernikus/master/assets/images/docs/containers/kubernetes/userroleassignments.png)

To add additional users to a cluster they need to be
given either `Kubernetes Admin` or `Kubernetes Member` roles.

![Role Assignments](https://raw.githubusercontent.com/sapcc/kubernikus/master/assets/images/docs/containers/kubernetes/roleassignment.png)

## Authenticating with Kubernetes

Kubernetes is usually remote controlled with the `kubectl` command line tool.
It is configured through a config file `.kubeconfig`. For installation
instructions please see the [official
documentation](https://kubernetes.io/docs/user-guide/kubectl-overview/).

### Manual Download
A preconfigured `.kubeconfig` file can be downloaded from the UI or fetched via
API:

![Download Credentials](https://raw.githubusercontent.com/sapcc/kubernikus/master/assets/images/docs/containers/kubernetes/credentials.png)

### Automatic Refresh

Since the certificates expire daily it becomes quite tedious to download new
`.kubeconfig` files every day. To help with this workflow there is a CLI tool
`kubernikusctl` for remote controlling Kubernikus clusters.

The tool is available precompiled for Linux, MacOS and Windows. The latest
version can be downloaded from [Github](https://github.com/sapcc/kubernikus/releases/latest).

Setting up an automatic refresh of the `.kubeconfig` file is a 2-step process:

  1. `kubernikusctl auth init --many --auth --options`
  2. `kubernikusctl auth refresh`

The initialisation only needs to be done once. Afterwards a `refresh` is
possible without repeating all authentication details.

![Setup](https://raw.githubusercontent.com/sapcc/kubernikus/master/assets/images/docs/containers/kubernetes/setup.png)

The UI provides the full `kubernikusctl auth init` initialisation command for
convenience.

### Default Permissions

By default any user with the `Kubernetes Admin` OpenStack role is assigned the
`cluster-admin` Kubernetes role. This is a super-admin that is allowed
everything.

Otherwise, the default Kubernetes RBAC policies grant very restrictive
permissions. Users with the `Kubernetes Member` OpenStack role need to be
assigned further permissions.

For example, to grant cluster-wide, full access:

```
kubectl create clusterrolebinding cluster-admin
  --clusterrole=cluster-admin
  --group=kubernetes_member
```

~> Note: This allows to perform any action against the API, including viewing secrets and modifying permissions. It is not a recommended policy.


