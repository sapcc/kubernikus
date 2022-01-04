package migration

import (
	"fmt"
	"strconv"

	helm_2to3 "github.com/helm/helm-2to3/pkg/v3"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/helm"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

func Helm2to3(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) error {
	return migrateHelmReleases(current, clients)
}

func migrateHelmReleases(kluster *v1.Kluster, clients config.Clients) error {
	// Implements `helm2to3 convert` roughly
	// https://github.com/helm/helm-2to3/blob/927e49f49fb04a562a3e14d9ada073ca61d21e7c/cmd/convert.go#L106
	versions2, err := getHelm2ReleaseVersions(kluster.Name, clients)
	if err != nil {
		return err
	}
	client2 := clients.Helm
	client3 := clients.Helm3
	for _, version2 := range versions2 {
		rsp, err := client2.ReleaseContent(kluster.Name, helm.ContentReleaseVersion(int32(version2)))
		if err != nil {
			return err
		}
		release3, err := helm_2to3.CreateRelease(rsp.Release)
		if err != nil {
			return err
		}
		err = client3.Releases.Create(release3)
		if err != nil {
			return err
		}
	}
	return nil
}

func getHelm2ReleaseVersions(releaseName string, clients config.Clients) ([]int, error) {
	configMaps, err := clients.Kubernetes.CoreV1().ConfigMaps("kube-system").List(meta_v1.ListOptions{
		LabelSelector: fmt.Sprintf("OWNER=TILLER,NAME=%s", releaseName),
	})
	if err != nil {
		return nil, err
	}
	versions := make([]int, 0)
	for _, configMap := range configMaps.Items {
		versionStr := configMap.Labels["VERSION"]
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, err
}
