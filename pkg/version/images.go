package version

import (
	"errors"
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type ImageVersion struct {
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
}

func (v ImageVersion) String() string {
	if v.Tag == "" {
		return v.Repository
	}
	return v.Repository + ":" + v.Tag
}

type KlusterVersion struct {
	Default                bool         `yaml:"default"`
	Supported              bool         `yaml:"supported"`
	Hyperkube              ImageVersion `yaml:"hyperkube,omitempty"`
	CloudControllerManager ImageVersion `yaml:"cloudControllerManager"`
	Dex                    ImageVersion `yaml:"dex,omitempty"`
	Dashboard              ImageVersion `yaml:"dashboard,omitempty"`
	DashboardProxy         ImageVersion `yaml:"dashboardProxy,omitempty"`
	Apiserver              ImageVersion `yaml:"apiserver,omitempty"`
	Scheduler              ImageVersion `yaml:"scheduler,omitempty"`
	ControllerManager      ImageVersion `yaml:"controllerManager,omitempty"`
	Kubelet                ImageVersion `yaml:"kubelet,omitempty"`
	KubeProxy              ImageVersion `yaml:"kubeProxy,omitempty"`
}

type ImageRegistry struct {
	Versions       map[string]KlusterVersion `yaml:"imagesForVersion,omitempty"`
	DefaultVersion string                    `yaml:"-"`
}

func NewImageRegistry(filepath string) (*ImageRegistry, error) {

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
