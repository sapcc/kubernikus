package ground

import "k8s.io/apimachinery/pkg/runtime/schema"

// Maintaining this list is important to find orphaned
// resource during the seeding reconciliation. GVK's not
// present on an API server are ignored.
var managedGVKs = []schema.GroupVersionKind{
	{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "ClusterRole",
	},
	{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "ClusterRoleBinding",
	},
	{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "Role",
	},
	{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "RoleBinding",
	},
	{
		Group:   "storage.k8s.io/v1",
		Version: "v1",
		Kind:    "StorageClass",
	},
	{
		Group:   "snapshot.storage.k8s.io/v1",
		Version: "v1",
		Kind:    "VolumeSnapshotClass",
	},
	{
		Group:   "storage.k8s.io/v1",
		Version: "v1",
		Kind:    "CSIDriver",
	},
	{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	},
	{
		Group:   "apps",
		Version: "v1",
		Kind:    "DaemonSet",
	},
	{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	},
	{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	},
	{
		Group:   "",
		Version: "v1",
		Kind:    "ServiceAccount",
	},
}
