package ground

import (
	"bytes"
	"context"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/releaseutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
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

type objectDiff struct {
	planned  unstructured.Unstructured
	deployed *unstructured.Unstructured
}

type SeedReconciler struct {
	Clients *config.Clients
	Kluster *v1.Kluster
	Logger  log.Logger
}

func (sr *SeedReconciler) EnrichHelmValuesForSeed(client project.ProjectClient, values map[string]interface{}) error {
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
	_, err = k8sClient.ExtensionsV1beta1().Deployments("kube-system").Get(context.TODO(), "kube-dns", metav1.GetOptions{})
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
		"count", len(planned),
		"v", 6)

	groupRessources, err := restmapper.GetAPIGroupResources(discover)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupRessources)
	client, err := sr.Clients.Satellites.DynamicClientFor(sr.Kluster)
	if err != nil {
		return err
	}
	diffs, err := getDiffObjects(client, mapper, planned)
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
		"kluster", sr.Kluster.Name,
		"v", 5)
	return nil
}

// Gets all resources as rendered by the seed chart
func getPlannedObjects(config *rest.Config, kubeVersion *version.Info, apiVersions chartutil.VersionSet, chartPath string, values map[string]interface{}) ([]unstructured.Unstructured, error) {
	planned := make([]unstructured.Unstructured, 0)
	seedChart, err := loader.Load(chartPath)
	if err != nil {
		return planned, err
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
		return planned, err
	}
	rendered, err := engine.RenderWithClient(seedChart, renderValues, config)
	if err != nil {
		return planned, err
	}
	_, manifests, err := releaseutil.SortManifests(rendered, apiVersions, releaseutil.InstallOrder)
	if err != nil {
		return planned, err
	}

	crds := make([][]byte, 0)
	for _, crdObject := range seedChart.CRDObjects() {
		crds = append(crds, crdObject.File.Data)
	}
	zipped := make([][]byte, 0)
	zipped = append(zipped, crds...)
	for _, manifest := range manifests {
		zipped = append(zipped, []byte(manifest.Content))
	}
	for _, manifest := range zipped {
		decoded := make(map[string]interface{})
		err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(manifest)), 1024).Decode(&decoded)
		if err != nil {
			return planned, err
		}
		obj := unstructured.Unstructured{Object: decoded}
		labels := obj.GetLabels()
		if len(labels) == 0 {
			labels = make(map[string]string)
		}
		labels[ManagedByLabelKey] = ManagedByLabelValue
		obj.SetLabels(labels)
		planned = append(planned, obj)
	}
	return planned, nil
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
		for _, oneManaged := range managedList.Items {
			managed = append(managed, oneManaged)
		}
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
			"kind", fmt.Sprintf("%s", oneOrphaned.GetKind()),
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
		if err != nil {
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
		"kind", fmt.Sprintf("%s", obj.GetKind()),
		"v", 6)
	_, err := makeScopedClient(client, mapping, obj.GetNamespace()).Create(context.TODO(), obj, metav1.CreateOptions{})
	return err
}

func (sr *SeedReconciler) patchDeployed(client dynamic.Interface, mapping *meta.RESTMapping, planned, deployed *unstructured.Unstructured) error {
	// skip if flagged
	if val, ok := deployed.GetLabels()[SkipPatchKey]; ok && val == SkipPatchValue {
		return nil
	}
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
	// Depending on the concrete resource there still patches that are not strictly
	// required fallthrough here. A prime example is the Container Spec of Deployments,
	// DaemonSets and so on, which has a bunch of optional fields, which aren't part
	// of the planned maifest but of the deployed resources. That in turn creates some
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

	sr.Logger.Log(
		"msg", "Seed reconciliation: patching deployed",
		"name", deployed.GetName(),
		"namespace", deployed.GetNamespace(),
		"kind", fmt.Sprintf("%s", deployed.GetKind()),
		"patch", string(patch),
		"v", 6)
	_, err = makeScopedClient(client, mapping, deployed.GetNamespace()).Patch(context.TODO(), deployed.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}
