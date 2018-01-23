Kubernikus Controllers
======================
The Kubernikus operator contains various independent controllers each one handling a different aspect of the managed kubernetes clusters.


Route garbage collector
-----------------------
The `routegc` controller is a stopgap solution to mitigate a bug in the official kubernetes OpenStack cloud provider (See [kubernetes/kubernetes#56258](https://github.com/kubernetes/kubernetes/pull/56258)).
The bug occasionally leaves static route entries in the cluster's OpenStack router when a Node is deleted. This causes routing problems when a new Node is reusing the network segment for Pods.

The bug is fixed in the upcoming 1.10 version of kubernetes. That means this controller is only needed for clusters <1.10.

For each cluster the controller polls the corresponding OpenStack router and inspects the configured static routes. It tries to identify and remove orphaned routes to fix the clusters networking. It removes all route entries for CIDR ranges that are within the clusters CIDR range for Pods where the target/nextHop ip address can't be matched to an OpenStack compute instance.
