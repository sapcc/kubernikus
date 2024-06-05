package ground

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/releaseutil"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack/project"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

const SeedChartPath string = "charts/seed"

const ManagedByLabelKey string = "cloud.sap/managed-by"
const ManagedByLabelValue string = "kubernikus"
const SkipPatchKey string = "kubernikus.cloud.sap/skip-manage"
const SkipPatchValue string = "true"

var recreateKinds map[string]struct{} = map[string]struct{}{
	"RoleBinding":        {},
	"ClusterRoleBinding": {},
	"StorageClass":       {},
}

type objectDiff struct {
	planned  unstructured.Unstructured
	deployed *unstructured.Unstructured
}

type plannedObjects struct {
	crds  []unstructured.Unstructured
	other []unstructured.Unstructured
}

type SeedReconciler struct {
	Clients *config.Clients
	Kluster *v1.Kluster
	Logger  log.Logger
}

func (sr *SeedReconciler) EnrichHelmValuesForSeed(client project.ProjectClient, values map[string]interface{}, kluster *v1.Kluster, secret *v1.Secret) error {
	metadata, err := client.GetMetadata()
	if err != nil {
		return err
	}
	azNames := make([]interface{}, 0)
	for _, az := range metadata.AvailabilityZones {
		azNames = append(azNames, az.Name)
	}
	if osValues, ok := values["openstack"]; ok {
		casted := osValues.(map[string]interface{})
		casted["azs"] = azNames
	}

	k8sClient, err := sr.Clients.Satellites.ClientFor(sr.Kluster)
	if err != nil {
		return err
	}
	// required to adapat old kube-dns deployments
	_, err = k8sClient.AppsV1().Deployments("kube-system").Get(context.TODO(), "kube-dns", metav1.GetOptions{})
	var isKubeDns bool
	if err == nil {
		isKubeDns = true
	} else if errors.IsNotFound(err) {
		isKubeDns = false
	} else {
		return err
	}
	values["dns"] = map[string]interface{}{
		"address": sr.Kluster.Spec.DNSAddress,
		"domain":  sr.Kluster.Spec.DNSDomain,
		"kube":    isKubeDns,
	}
	values["customCNI"] = kluster.Spec.CustomCNI
	values["seedKubeadm"] = kluster.Spec.SeedKubeadm
	values["seedVirtual"] = kluster.Spec.SeedVirtual
	idx := strings.LastIndex(kluster.Spec.Name, "-")
	if idx != -1 {
		values["shortName"] = kluster.Spec.Name[:idx]
	}
	values["tlsCaCert"] = secret.TLSCACertificate
	values["kubeletClientsCaCert"] = secret.KubeletClientsCACertificate
	return nil
}

func NewSeedReconciler(clients *config.Clients, kluster *v1.Kluster, logger log.Logger) SeedReconciler {
	return SeedReconciler{Clients: clients, Kluster: kluster, Logger: logger}
}

func (sr *SeedReconciler) ReconcileSeeding(chartPath string, values map[string]interface{}) error {
	config, err := sr.Clients.Satellites.ConfigFor(sr.Kluster)
	if err != nil {
		return err
	}
	discover, err := discovery.NewDiscoveryClientForConfig(&config)
	if err != nil {
		return err
	}
	apiVersions, err := action.GetVersionSet(discover)
	if err != nil {
		return err
	}

	version, err := discover.ServerVersion()
	if err != nil {
		return err
	}
	planned, err := getPlannedObjects(&config, version, apiVersions, chartPath, values)
	if err != nil {
		return err
	}
	sr.Logger.Log(
		"msg", "Seed reconciliation: planned objects",
		"crds", len(planned.crds),
		"other", len(planned.other),
		"kluster", sr.Kluster.GetName(),
		"project", sr.Kluster.Account(),
		"v", 6)

	groupRessources, err := restmapper.GetAPIGroupResources(discover)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupRessources)
	// it is required to deal with CRDs first, because otherwise creation of CDRs instances
	// just to be created, will fail due to caching in the RESTMapper.
	if len(planned.crds) > 0 {
		client, err := sr.Clients.Satellites.DynamicClientFor(sr.Kluster)
		if err != nil {
			return err
		}
		diffs, err := getDiffObjects(client, mapper, planned.crds)
		if err != nil {
			return err
		}
		managed, err := getCrdObjects(sr.Clients, mapper, sr.Kluster)
		if err != nil {
			return err
		}
		orphaned := findOrphanedObjects(diffs, managed)
		err = sr.deleteOrphanedObjects(client, mapper, orphaned)
		if err != nil {
			return err
		}
		err = sr.createOrUpdateObjects(client, mapper, diffs)
		if err != nil {
			return err
		}
		sr.Logger.Log(
			"msg", "Seed reconciliation: reconciled CRDs",
			"kluster", sr.Kluster.GetName(),
			"project", sr.Kluster.Account(),
			"v", 5)
		// Recreate discovery and RESTMapper client so it knows about new CRDs.
		// The old does not have changed CRDs chached.
		groupRessources, err = restmapper.GetAPIGroupResources(discover)
		if err != nil {
			return err
		}
		mapper = restmapper.NewDiscoveryRESTMapper(groupRessources)
	}

	client, err := sr.Clients.Satellites.DynamicClientFor(sr.Kluster)
	if err != nil {
		return err
	}
	diffs, err := getDiffObjects(client, mapper, planned.other)
	if err != nil {
		return err
	}
	managed, err := getManagedObjects(sr.Clients, mapper, sr.Kluster)
	if err != nil {
		return err
	}
	orphaned := findOrphanedObjects(diffs, managed)
	err = sr.deleteOrphanedObjects(client, mapper, orphaned)
	if err != nil {
		return err
	}
	err = sr.createOrUpdateObjects(client, mapper, diffs)
	if err != nil {
		return err
	}
	sr.Logger.Log(
		"msg", "Seed reconciliation: successful",
		"kluster", sr.Kluster.GetName(),
		"project", sr.Kluster.Account(),
		"v", 5)
	return nil
}

// Gets all resources as rendered by the seed chart
func getPlannedObjects(config *rest.Config, kubeVersion *version.Info, apiVersions chartutil.VersionSet, chartPath string, values map[string]interface{}) (plannedObjects, error) {
	seedChart, err := loader.Load(chartPath)
	if err != nil {
		return plannedObjects{}, err
	}
	renderValues, err := chartutil.ToRenderValues(seedChart, values, chartutil.ReleaseOptions{}, &chartutil.Capabilities{
		APIVersions: apiVersions,
		KubeVersion: chartutil.KubeVersion{
			Version: kubeVersion.GitVersion,
			Major:   kubeVersion.Major,
			Minor:   kubeVersion.Minor,
		},
	})
	if err != nil {
		return plannedObjects{}, err
	}
	rendered, err := engine.RenderWithClient(seedChart, renderValues, config)
	if err != nil {
		return plannedObjects{}, err
	}
	_, manifests, err := releaseutil.SortManifests(rendered, apiVersions, releaseutil.InstallOrder)
	if err != nil {
		return plannedObjects{}, err
	}

	crds := make([]unstructured.Unstructured, 0)
	for _, crdObject := range seedChart.CRDObjects() {
		decoded, err := decodeUnstructured(crdObject.File.Data)
		if err != nil {
			return plannedObjects{}, err
		}
		crds = append(crds, decoded)
	}
	other := make([]unstructured.Unstructured, 0)
	for _, manifest := range manifests {
		decoded, err := decodeUnstructured([]byte(manifest.Content))
		if err != nil {
			return plannedObjects{}, err
		}
		other = append(other, decoded)
	}
	return plannedObjects{
		crds:  crds,
		other: other,
	}, nil
}

func decodeUnstructured(manifest []byte) (unstructured.Unstructured, error) {
	decoded := make(map[string]interface{})
	err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(manifest)), 1024).Decode(&decoded)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	obj := unstructured.Unstructured{Object: decoded}
	labels := obj.GetLabels()
	if len(labels) == 0 {
		labels = make(map[string]string)
	}
	labels[ManagedByLabelKey] = ManagedByLabelValue
	obj.SetLabels(labels)
	return obj, nil
}

// Gets all resources that are managed by kubernikus from the cluster
func getManagedObjects(clients *config.Clients, mapper meta.RESTMapper, kluster *v1.Kluster) ([]unstructured.Unstructured, error) {
	dynamicClient, err := clients.Satellites.DynamicClientFor(kluster)
	if err != nil {
		return nil, err
	}
	managed := make([]unstructured.Unstructured, 0)
	for _, gvk := range managedGVKs {
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if meta.IsNoMatchError(err) {
			// lets assume the given gvk was valid but removed in recent kubernetes version
			continue
		} else if err != nil {
			return nil, err
		}
		managedList, err := makeScopedClient(dynamicClient, mapping, "kube-system").List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", ManagedByLabelKey, ManagedByLabelValue)})
		if err != nil {
			return nil, err
		}
		managed = append(managed, managedList.Items...)
	}
	return managed, nil
}

// Gets all CRDS that are managed by kubernikus from the cluster
func getCrdObjects(clients *config.Clients, mapper meta.RESTMapper, kluster *v1.Kluster) ([]unstructured.Unstructured, error) {
	dynamicClient, err := clients.Satellites.DynamicClientFor(kluster)
	if err != nil {
		return nil, err
	}
	managedCRDs := []schema.GroupVersionKind{
		{
			Group:   "apiextensions.k8s.io/v1",
			Version: "v1",
			Kind:    "CustomResourceDefinition",
		},
	}
	managed := make([]unstructured.Unstructured, 0)
	for _, gvk := range managedCRDs {
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if meta.IsNoMatchError(err) {
			// lets assume the given gvk was valid but removed in recent kubernetes version
			continue
		} else if err != nil {
			return nil, err
		}
		managedList, err := makeScopedClient(dynamicClient, mapping, "kube-system").List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", ManagedByLabelKey, ManagedByLabelValue)})
		if err != nil {
			return nil, err
		}
		managed = append(managed, managedList.Items...)
	}
	return managed, nil
}

func makeScopedClient(client dynamic.Interface, mapping *meta.RESTMapping, namespace string) dynamic.ResourceInterface {
	if mapping.Scope.Name() == meta.RESTScopeNameRoot {
		return client.Resource(mapping.Resource)
	}
	return client.Resource(mapping.Resource).Namespace(namespace)
}

// Takes the planned resources and fetches the remote version if it exists
func getDiffObjects(client dynamic.Interface, mapper meta.RESTMapper, planned []unstructured.Unstructured) ([]objectDiff, error) {
	diffs := make([]objectDiff, 0)
	for _, onePlanned := range planned {
		mapping, err := mapper.RESTMapping(onePlanned.GroupVersionKind().GroupKind(), onePlanned.GroupVersionKind().Version)
		if meta.IsNoMatchError(err) {
			// a planned instace of a CRD could come along here, which cannot be found, as the CRD has not been created yet
			continue
		} else if err != nil {
			return nil, err
		}
		oneDeployed, err := makeScopedClient(client, mapping, onePlanned.GetNamespace()).Get(context.TODO(), onePlanned.GetName(), metav1.GetOptions{})
		if errors.IsNotFound(err) {
			oneDeployed = nil
		} else if err != nil {
			return diffs, err
		}
		diffs = append(diffs, objectDiff{
			planned:  onePlanned,
			deployed: oneDeployed,
		})
	}
	return diffs, nil
}

// Returns objects which are currently managed but are not planned anymore
// Remark: the diffs depend on the planned resources
func findOrphanedObjects(diffs []objectDiff, managed []unstructured.Unstructured) []unstructured.Unstructured {
	orphaned := make([]unstructured.Unstructured, 0)
	for _, oneManaged := range managed {
		if !diffsContain(diffs, &oneManaged) {
			orphaned = append(orphaned, oneManaged)
		}
	}
	return orphaned
}

func diffsContain(diffs []objectDiff, obj *unstructured.Unstructured) bool {
	for _, oneDiff := range diffs {
		if oneDiff.planned.GetName() == obj.GetName() &&
			oneDiff.planned.GetNamespace() == obj.GetNamespace() &&
			oneDiff.planned.GroupVersionKind() == obj.GroupVersionKind() {
			return true
		}
	}
	return false
}

func (sr *SeedReconciler) deleteOrphanedObjects(client dynamic.Interface, mapper meta.RESTMapper, orphans []unstructured.Unstructured) error {
	for _, oneOrphaned := range orphans {
		sr.Logger.Log(
			"msg", "Seed reconciliation: deleting orphaned",
			"name", oneOrphaned.GetName(),
			"namespace", oneOrphaned.GetNamespace(),
			"kind", oneOrphaned.GetKind(),
			"kluster", sr.Kluster.GetName(),
			"project", sr.Kluster.Account(),
			"v", 6)
		mapping, err := mapper.RESTMapping(oneOrphaned.GroupVersionKind().GroupKind(), oneOrphaned.GroupVersionKind().Version)
		if err != nil {
			return err
		}
		err = makeScopedClient(client, mapping, oneOrphaned.GetNamespace()).Delete(context.TODO(), oneOrphaned.GetName(), metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (sr *SeedReconciler) createOrUpdateObjects(client dynamic.Interface, mapper meta.RESTMapper, diffs []objectDiff) error {
	for _, oneDiff := range diffs {
		mapping, err := mapper.RESTMapping(oneDiff.planned.GroupVersionKind().GroupKind(), oneDiff.planned.GroupVersionKind().Version)
		if meta.IsNoMatchError(err) {
			// a planned instace of a CRD could come along here, which cannot be found, as the CRD has not been created yet
			continue
		} else if err != nil {
			return err
		}
		if oneDiff.deployed == nil {
			err = sr.createPlanned(client, mapping, &oneDiff.planned)
			if err != nil {
				return err
			}
		} else {
			err = sr.patchDeployed(client, mapping, &oneDiff.planned, oneDiff.deployed)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (sr *SeedReconciler) createPlanned(client dynamic.Interface, mapping *meta.RESTMapping, obj *unstructured.Unstructured) error {
	sr.Logger.Log(
		"msg", "Seed reconciliation: creating planned",
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
		"kind", obj.GetKind(),
		"kluster", sr.Kluster.GetName(),
		"project", sr.Kluster.Account(),
		"v", 6)
	_, err := makeScopedClient(client, mapping, obj.GetNamespace()).Create(context.TODO(), obj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	if obj.GroupVersionKind().Kind == "CustomResourceDefinition" {
		sr.Logger.Log(
			"msg", "Seed reconciliation: awaiting crd established",
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", obj.GetKind(),
			"kluster", sr.Kluster.GetName(),
			"project", sr.Kluster.Account(),
			"v", 6)
		return wait.Poll(500*time.Millisecond, 20*time.Second, func() (done bool, err error) { //nolint:staticcheck
			crd, err := makeScopedClient(client, mapping, obj.GetNamespace()).Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			status := crd.Object["status"].(map[string]interface{})
			conditions := status["conditions"].([]interface{})
			for _, condition := range conditions {
				c := condition.(map[string]interface{})
				ty := c["type"].(string)
				statusStr := c["status"].(string)
				if ty == string(extensionsv1.Established) && statusStr == string(metav1.ConditionTrue) {
					return true, nil
				}
			}
			return false, nil
		})
	}
	return nil
}

func (sr *SeedReconciler) patchDeployed(client dynamic.Interface, mapping *meta.RESTMapping, planned, deployed *unstructured.Unstructured) error {
	// skip if flagged
	if val, ok := deployed.GetLabels()[SkipPatchKey]; ok && val == SkipPatchValue {
		return nil
	}
	// make an unmodified deep copy of the planned object, which we could need re-creation
	plannedCopy := planned.DeepCopy()
	// copy over certain values to keep patches small
	deployedMetadata := deployed.Object["metadata"].(map[string]interface{})
	plannedMetadata := planned.Object["metadata"].(map[string]interface{})
	plannedMetadata["creationTimestamp"] = deployedMetadata["creationTimestamp"]
	plannedMetadata["managedFields"] = deployedMetadata["managedFields"]
	plannedMetadata["resourceVersion"] = deployedMetadata["resourceVersion"]
	plannedMetadata["uid"] = deployedMetadata["uid"]
	if _, ok := deployed.Object["status"]; ok {
		planned.Object["status"] = deployed.Object["status"]
	}
	if _, ok := deployed.Object["reclaimPolicy"]; ok {
		planned.Object["reclaimPolicy"] = deployed.Object["reclaimPolicy"]
	}
	// ServiceAccounts have a top level secret field, which contains a reference to
	// the token. Clearing that causes a new token to be created in versions below
	// 1.24 filling up the controlplane. That field is no longer present in newer
	// versions.
	if deployed.GetKind() == "ServiceAccount" {
		if secrets, ok := deployed.Object["secrets"]; ok {
			planned.Object["secrets"] = secrets
		}
	}
	// Depending on the concrete resource there still patches that are not strictly
	// required fallthrough here. A prime example is the Container Spec of Deployments,
	// DaemonSets and so on, which has a bunch of optional fields, which aren't part
	// of the planned manifest but of the deployed resources. That in turn creates some
	// larger patches.

	original, err := deployed.MarshalJSON()
	if err != nil {
		return err
	}
	modified, err := planned.MarshalJSON()
	if err != nil {
		return err
	}
	patch, err := jsonpatch.CreateMergePatch(original, modified)
	if err != nil {
		return err
	}
	if string(patch) == "{}" {
		return nil
	}

	// try to apply patch first
	err = sr.patchResource(client, mapping, deployed, patch)
	if err == nil {
		return nil
	} else if errors.IsInvalid(err) {
		// if the patch is invalid we can try to recreate certain resources
		if recreateAllowed(deployed.GetKind()) {
			return sr.recreateResource(client, mapping, plannedCopy)
		}
	}
	return err
}

func recreateAllowed(kind string) bool {
	_, ok := recreateKinds[kind]
	return ok
}

func (sr *SeedReconciler) patchResource(client dynamic.Interface, mapping *meta.RESTMapping, deployed *unstructured.Unstructured, patch []byte) error {
	sr.Logger.Log(
		"msg", "Seed reconciliation: patching deployed",
		"name", deployed.GetName(),
		"namespace", deployed.GetNamespace(),
		"kind", deployed.GetKind(),
		"patch", string(patch),
		"kluster", sr.Kluster.GetName(),
		"project", sr.Kluster.Account(),
		"v", 6)
	_, err := makeScopedClient(client, mapping, deployed.GetNamespace()).Patch(context.TODO(), deployed.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}

func (sr *SeedReconciler) recreateResource(client dynamic.Interface, mapping *meta.RESTMapping, planned *unstructured.Unstructured) error {
	sr.Logger.Log(
		"msg", "Seed reconciliation: recreating deployed",
		"name", planned.GetName(),
		"namespace", planned.GetNamespace(),
		"kind", planned.GetKind(),
		"kluster", sr.Kluster.GetName(),
		"project", sr.Kluster.Account(),
		"v", 6)
	// refuse to delete any resource called kubernikus:admin.
	// this could delete the clusterrolebinding we need to get
	// into the cluster and lock ourselves out
	if planned.GetName() == "kubernikus:admin" {
		return fmt.Errorf("refusing to recreate a resource with name kubernikus:admin")
	}
	scoped := makeScopedClient(client, mapping, planned.GetNamespace())
	err := scoped.Delete(context.TODO(), planned.GetName(), metav1.DeleteOptions{})
	if err != nil {
		return nil
	}
	_, err = scoped.Create(context.TODO(), planned, metav1.CreateOptions{})
	return err
}
