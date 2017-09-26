package main

import (
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	APIURL               string `yaml:"kubernikus_api_server"`
	APIVersion           string `yaml:"kubernikus_api_version"`
	KubeConfig           string `yaml:"kluster_kubeconfig"`
	OpenStackCredentials `yaml:"openstack"`
}

func ReadConfig(filePath string) (Config, error) {
	var cfg Config
	cfgBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return cfg, fmt.Errorf("read configuration file: %s", err.Error())
	}
	err = yaml.Unmarshal(cfgBytes, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("parse configuration: %s", err.Error())
	}

	return cfg, nil
}

func ReadFromEnv() (Config, error) {
	return Config{
		APIURL:     os.Getenv("KUBERNIKUS_API_SERVER"),
		APIVersion: os.Getenv("KUBERNIKUS_API_VERSION"),
	}, nil
}

func (cfg *Config) Verify() error {
	if cfg.APIURL == "" {
		cfg.APIURL = "kubernikus.staging.cloud.sap"
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = "v1"
	}
	return cfg.OpenStackCredentials.Verify()
}
