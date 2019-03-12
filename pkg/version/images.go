package version

import (
	"errors"
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type KlusterVersion struct {
	Default   bool   `yaml:"default"`
	Hyperkube string `yaml:"hyperkube"`
}

type ImageRegistry struct {
	Versions       map[string]KlusterVersion `yaml:"imagesForVersion"`
	DefaultVersion string
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
		return nil, fmt.Errorf("No versions found in %s", file)
	}
	for v, info := range registry.Versions {
		if info.Default {
			if registry.DefaultVersion != "" {
				return nil, fmt.Errorf("Multiple default versions found: %s and %s", registry.DefaultVersion, v)
			}
			registry.DefaultVersion = v
		}
	}
	if registry.DefaultVersion == "" {
		return nil, errors.New("No default version specified")
	}

	return registry, nil

}
