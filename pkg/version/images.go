package version

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type ImageVersion struct {
	Repository string `yaml:"repository" json:"repository"`
	Tag        string `yaml:"tag" json:"tag"`
}

func (v ImageVersion) String() string {
	if v.Tag == "" {
		return v.Repository
	}
	return v.Repository + ":" + v.Tag
}

type KlusterVersion struct {
	Default                bool         `yaml:"default" json:"default"`
	Supported              bool         `yaml:"supported" json:"supported"`
	Hyperkube              ImageVersion `yaml:"hyperkube,omitempty" json:"hyperkube,omitempty"`
	CloudControllerManager ImageVersion `yaml:"cloudControllerManager,omitempty" json:"cloudControllerManager,omitempty"`
	Dex                    ImageVersion `yaml:"dex,omitempty" json:"dex,omitempty"`
	Dashboard              ImageVersion `yaml:"dashboard,omitempty" json:"dashboard,omitempty"`
	DashboardProxy         ImageVersion `yaml:"dashboardProxy,omitempty" json:"dashboardProxy,omitempty"`
	Apiserver              ImageVersion `yaml:"apiserver,omitempty" json:"apiserver,omitempty"`
	Scheduler              ImageVersion `yaml:"scheduler,omitempty" json:"scheduler,omitempty"`
	ControllerManager      ImageVersion `yaml:"controllerManager,omitempty" json:"controllerManager,omitempty"`
	Kubelet                ImageVersion `yaml:"kubelet,omitempty" json:"kubelet,omitempty"`
	KubeProxy              ImageVersion `yaml:"kubeProxy,omitempty" json:"kubeProxy,omitempty"`
	CoreDNS                ImageVersion `yaml:"coreDNS,omitempty" json:"coreDNS,omitempty"`
	Pause                  ImageVersion `yaml:"pause,omitempty" json:"pause,omitempty"`
	Wormhole               ImageVersion `yaml:"wormhole,omitempty" json:"wormhole,omitempty"`
	Etcd                   ImageVersion `yaml:"etcd,omitempty" json:"etcd,omitempty"`
	EtcdBackup             ImageVersion `yaml:"etcdBackup,omitempty" json:"etcdBackup,omitempty"`
	CSIAttacher            ImageVersion `yaml:"csiAttacher,omitempty" json:"csiAttacher,omitempty"`
	CSIProvisioner         ImageVersion `yaml:"csiProvisioner,omitempty" json:"csiProvisioner,omitempty"`
	CSISnapshotter         ImageVersion `yaml:"csiSnapshotter,omitempty" json:"csiSnapshotter,omitempty"`
	CSISnapshotController  ImageVersion `yaml:"csiSnapshotController,omitempty" json:"csiSnapshotController,omitempty"`
	CSIResizer             ImageVersion `yaml:"csiResizer,omitempty" json:"csiResizer,omitempty"`
	CSILivenessProbe       ImageVersion `yaml:"csiLivenessProbe,omitempty" json:"csiLivenessProbe,omitempty"`
	CSINodeDriver          ImageVersion `yaml:"csiNodeDriver,omitempty" json:"csiNodeDriver,omitempty"`
	CinderCSIPlugin        ImageVersion `yaml:"cinderCSIPlugin,omitempty" json:"cinderCSIPlugin,omitempty"`
	Flannel                ImageVersion `yaml:"flannel,omitempty" json:"flannel,omitempty"`
	Fluentd                ImageVersion `yaml:"fluentd,omitempty" json:"fluentd,omitempty"`
}

type ImageRegistry struct {
	Versions       map[string]KlusterVersion `yaml:"imagesForVersion,omitempty" json:"imagesForVersion,omitempty"`
	DefaultVersion string                    `yaml:"-" json:"-"`
}

func NewImageRegistry(filepath string, region string) (*ImageRegistry, error) {

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	registry := new(ImageRegistry)
	if err := yaml.NewDecoder(file).Decode(&registry); err != nil {
		return nil, err
	}
	if len(registry.Versions) < 1 {
		return nil, fmt.Errorf("No versions found in %s", filepath)
	}

	replaceRegionVarInRepositoryField(registry.Versions, region)

	for v, info := range registry.Versions {
		if info.Default {
			if registry.DefaultVersion != "" {
				return nil, fmt.Errorf("Multiple default versions found: %s and %s", registry.DefaultVersion, v)
			}
			registry.DefaultVersion = v
		}
		if info.Apiserver.Repository != "" {
			if info.Apiserver.Tag == "" {
				return nil, fmt.Errorf("Tag for apiserver image missing for version %s", v)
			}
			if info.ControllerManager.Repository == "" {
				return nil, fmt.Errorf("Repository for controller manager image missing for version %s", v)
			}
			if info.ControllerManager.Tag == "" {
				return nil, fmt.Errorf("Tag for controller manager image missing for version %s", v)
			}
			if info.Scheduler.Repository == "" {
				return nil, fmt.Errorf("Repository for scheduler image missing for version %s", v)
			}
			if info.Scheduler.Tag == "" {
				return nil, fmt.Errorf("Tag for scheduler image missing for version %s", v)
			}
			if info.Kubelet.Repository == "" {
				return nil, fmt.Errorf("Repository for kubelet image missing for version %s", v)
			}
			if info.Kubelet.Tag == "" {
				return nil, fmt.Errorf("Tag for kubelet image missing for version %s", v)
			}
			if info.KubeProxy.Repository == "" {
				return nil, fmt.Errorf("Repository for kube-proxy image missing for version %s", v)
			}
			if info.KubeProxy.Tag == "" {
				return nil, fmt.Errorf("Tag for kube-proxy image missing for version %s", v)
			}
			if info.Fluentd.Repository == "" {
				return nil, fmt.Errorf("Repository for fluentd image missing for version %s", v)
			}
			if info.Fluentd.Tag == "" {
				return nil, fmt.Errorf("Tag for fluentd image missing for version %s", v)
			}
		} else {
			if info.Hyperkube.Repository == "" {
				return nil, fmt.Errorf("Repository for hyperkube image missing for version %s", v)
			}
			if info.Hyperkube.Tag == "" {
				return nil, fmt.Errorf("Tag for hyperkube image missing for version %s", v)
			}
		}
	}
	if registry.DefaultVersion == "" {
		return nil, errors.New("No default version specified")
	}

	return registry, nil

}

func replaceRegionVarInRepositoryField(versions map[string]KlusterVersion, region string) {
	for i := range versions {
		v := versions[i]
		s := reflect.ValueOf(&v).Elem()
		for i := 0; i < s.NumField(); i++ {
			f := s.Field(i)
			if f.Type() == reflect.TypeOf(v.Hyperkube) {
				repo := f.FieldByName("Repository")
				repo.SetString(strings.Replace(repo.String(), "$REGION", region, 1))
			}
		}
		versions[i] = v
	}
}
