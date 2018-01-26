Kubernikus Controllers
======================
The Kubernikus operator contains various independent controllers each one handling a different aspect of the managed kubernetes clusters.

Deorbit Controller
------------------

The Kubernetes `controller-manager` runs controllers that are responsible for
creating, updating and deleting of OpenStack resources. It auto-provisions
Cinder volumes for Persistent Volume Claims, load balancers for services and
routes for nodes in the cluster.  When a kluster is terminated it is desired
that these resources are cleaned up automatically. 

The problem is that the `controller-manager` only cleans up, when the
respective resources are deleted from the cluster.

This is where the `deorbiter` comes into play. It inspects the cluster and
deletes all PVCs that are backed by Cinder, as well as all services of type
`LoadBalancer`. Afterwards, it waits until the `controller-manager` has
completed the clean-up.

Problematic is that Cinder volumes can take a variable time until they are
deleted. In some cases volumes might be stuck deleting indefinitely. The
`deorbiter` watches the PVs (note: PV not PVC) in the cluster and waits for all
of them to disappear. PVs are guarded by a finalizer and are only removed when
the cleanup completes. The `deorbiter` retries the deletion for a configurable
amount of time and eventually self-destructs the cluster. This might leave some
debris in the OpenStack project. 

For services Kubernetes does currently not implement the same finalizer
concept. That means that a service is deleted immediately while the clean-up
progresses in the background. Without double checking directly in OpenStack,
the `deboriter` can't tell when the deletion is completed. Here we assume
a fixed time, like 2 minutes, and hope for the best. 


Route garbage collector
-----------------------
The `routegc` controller is a stopgap solution to mitigate a bug in the official kubernetes OpenStack cloud provider (See [kubernetes/kubernetes#56258](https://github.com/kubernetes/kubernetes/pull/56258)).
The bug occasionally leaves static route entries in the cluster's OpenStack router when a Node is deleted. This causes routing problems when a new Node is reusing the network segment for Pods.

The bug is fixed in the upcoming 1.10 version of kubernetes. That means this controller is only needed for clusters <1.10.

For each cluster the controller polls the corresponding OpenStack router and inspects the configured static routes. It tries to identify and remove orphaned routes to fix the clusters networking. It removes all route entries for CIDR ranges that are within the clusters CIDR range for Pods where the target/nextHop ip address can't be matched to an OpenStack compute instance.


