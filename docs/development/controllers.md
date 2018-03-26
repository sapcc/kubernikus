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

For each cluster the controller polls the corresponding OpenStack router and
inspects the configured static routes. It tries to identify and remove orphaned
routes to fix the clusters networking. It removes all route entries for CIDR
ranges that are within the clusters CIDR range for Pods where the
target/nextHop ip address can't be matched to an OpenStack compute instance.


Flight Controller
-----------------

This controller takes care about Kluster health. It looks for obvious problems
and tries to repair them.

### Security Group (DVS) Update Latency

Updates to the security group are not instant. There is a non-trivial amount of
latency involved. This leads to timeouts and edge cases as the definition of
tolerable latency differs. Most notable this affects DHCP and Ignition. Default
tolerances are in the range of 30s.

The latency of updates is related to general load on the regions, "noisy"
neighbors blocking the update loops, amount of ports in the same project and
Neutron and VCenter performance.

If the updates take longer than these timeouts the following symptoms appear:

 * Nodes Running but never become healthy
 * No IPv4 Address visible on the VNC Console
 * No SSH Login Possible
 * Kubelet not running

These symptom indicates that the node couldn't configure its network
interface before the Ignition timeout. This effectifly leaves the node broken.

Possible Workarounds:

 * Increase DHCP/Ignition Timeout
	 This configuration needs to be baked into the image as an OEM customization.
	 It also interacts with the DHCP client timeout which again requires a
	 modification of the image. With frequent CoreOS updates this modification
	 needs to be automatic and included in the image build pipeline.

 * Reboot the Instance
	 This is the preferred workaround. It gives the DVS agents additional time to
	 configure the existing image and retries the Ignition run (to be verified).

 * Delete the Instance
	 This workaround is problematic. It will not succeed if the update latency is
	 too high in general.


### Security Group Update Event Missed

If an instance successfully downloads and startes the Kubelet, it registers itself
with the APIServer and gets a PodCIDR range assigned. This triggers a
reconfiguration of the Neutron Router and adds a static route. The route points
the PodCIDR to the node's IP address. This is required to satisfy the Kubernetes
pod to pod communication requirements.

As the PodCIDR subnet is now actually routed via the Neutron router it is required
to be allowed in the security group.

This happens by updating the node's Neutron port and adding the required CIDR to
`allowed_address_pairs`. This triggers an event that the port was updated. The DVS
agent are catching this update and adding an additional rule to the security group.

Occasionally, this update is missed. Until a full reconcilation loop happens
(usually by restarting or update of the DVS agents) the following symptoms appear:

 * Sporadic Pod Communication
 * Sporadic Service Communication
 * Sporadic Load Balancer Commnication
 * Sporadic DNS Problems in Pods
	 Depending on the disconnected node pods can't reach the Kube-DNS service. DNS
	 will work on the nodes.
 * Load Balancer Pools Flapping

Possible Workarounds:

 * Add PodCIDRs to Security Group
	 Instead of relying on the unreliable Oslo events, all possible PodCIDRs are
	 being added to the kluster's security group. Per default this is 198.19.0.0/16

 * Trigger Security Group Sync
	 Instead of waiting for a security group reconcilliation force an update by
	 periodially add a (random) change to the security group. If possible this
	 should only be triggered when a node condition indicates pod communication
	 problems.


###  Default Security Group Missing

When a new node is created via Nova a security group can be specified. The user
can select this security group during Kluster creation. If nothing is selected
the `default` security group is assumed.

For yet unknown reasons, there's a chance that the instance is configured without
this security group association by Nova. In effect the instance is completely
disconnected from the network.

Symptoms are similar to (1):

 * Nodes Running but never become healthy
 * No IPv4 Address visible on the VNC Console
 * No SSH Login Possible
 * Kubelet not running

Possible Workaround:

 * Reconcile Security Group Associations
	 Periodically check that instances which haven't registerd as nodes do have
	 the required security group enabled. If not, set it again.


### ASR Route Duplicates

When a node is being deleted its route is removed in Neutron. The ASR agents
get notified by an event and do remove the route from the ASR device.

First of all, this requires that the state of the Neutron DB reflects reality.
Updates to the routes are done by the RouteController in the Kubernetes OpenStack
cloud provider. Before 1.10 there's a bug that misses the updates. In Kubernikus
we fixed this by adding an additional RouteNanny for now.

When a Kluster is terminated forcefully, the RouteController might be destroyed
before it manages to update the Neutron database. The reconciliation happens
every 60 seconds. We counter this by gracefully deorbiting the Kluster waiting
for the updates either by the RouteController or the RouteNanny.

Unfortunately, during normal operations by scaling node pools or terminating nodes
updates to the routes do get missed as well. In that case the state in Neutron is
correct, while the state on the ASR device still shows the deleted route. This
should be fixable by triggering or manually running a re-sync. Unfortunately,
that does not work.

The re-sync mechanism between Neutron and ASR is not perfect. There is currently
the problem that routes that have been removed in Neutron will not be removed
during the sync. It only considers additional routes that have been added.

The only way to recover this situation is to manually `delete` and then `sync` the
router using the `asr1k_utils`. Additional node conditions that check for this
problem will facilitate alerting and manual intervention.

These dublicate routes are fatal because the IPAM module in the CNI plugin
recycles PodCIDRs immediately. A new node will receive the PodCIDR of a previously
existing node. The old node's routes are still configured on the ASR device and
take precedence. That leaves the new node broken. For example, the state of the
ASR routes after multiple node deletions:

	198.19.1.0/24 -> 10.180.0.3
	198.19.1.0/24 -> 10.180.0.4
	198.19.1.0/24 -> 10.180.0.5

Currently correct is the last route, pointing to the latest instance. In effect
is the first route pointing to 10.180.0.3 which doesn't exist anymore.

Symptoms:

 * See (2). Sporadic Pod/Service/LB Communication Problems
 * Only the first cluster in each project works

Workarounds:

 * None Known

 * There's no known way to trigger a router `asr1k_utils delete` +
	 `asr1k_utils sync` via OpenStack without actually deleting the whole Neutron
	 router construct. If a side-channel could somehow be leveraged it would be
	 possible to recover automatically.


### ASR Missing Routes

Due to various effects it is possible that the ASR agents miss the event to add
an additional route when a new node is created.

On specifically fatal effect is the failover between `ACTIVE` and `STANDBY`
router. It seems to be a rather common defect (potentially even intended) that
only the `ACTIVE` receives sync events. Upon failover routes are missing or
reflect the state of a previous cluster.

Symptoms:

 * See (2)

Workarounds:

 * Manual Sync

 * Trigger Sync automatically
	 There's no direct interface to trigger a sync via OpenStack API. It can be
	 forced indirectly by an action that triggers a sync: Attaching/Detaching a
	 FIP/Interface, Adding/Removing a Route


### Neutron Static Route Limit

There's a static limit of 31 routes in Neutron.

In projects with frequent Kluster create and deletes, the route limit can be
exceeded due duplicate routes as described in (4).

Symptoms:

 * See (2). Pod/Service/LB communication problems
 * 409 Conflict Errors in RouteController and RouteNanny
 * Klusters have a max size of 31 Nodes

Workarounds:

 * None. Neutron needs to be reconfigured
 * Clean up duplicate routes

