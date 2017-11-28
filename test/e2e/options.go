package main

import (
	"github.com/golang/glog"
)

type E2ETestSuiteOptions struct {
	Config
	ConfigFile string

	IsTestAll     bool
	IsTestAPI     bool
	IsTestSmoke   bool
	IsTestCreate  bool
	IsTestNetwork bool
	IsTestVolume  bool
	IsTestDelete  bool

	IsNoTeardown bool
}

func (o *E2ETestSuiteOptions) OptionsFromConfigFile() error {
	if o.ConfigFile == "" {
		o.ConfigFile = "test/e2e/e2e_config.yaml"
		glog.Infof("mandatory path to config not provided. trying default.")
	}

	cfg, err := ReadConfig(o.ConfigFile)
	if err != nil {
		return err
	}

	o.Config = cfg
	o.checkTestPhases()

	return nil
}

func (o *E2ETestSuiteOptions) checkTestPhases() {
	o.IsTestAll = false
	// if no phase is specified run the whole test suite
	if !o.isAnyPhaseSpecified() {
		o.IsTestAll = true
	}
}

func (o *E2ETestSuiteOptions) isAnyPhaseSpecified() bool {
	return o.IsTestCreate ||
		o.IsTestAPI ||
		o.IsTestSmoke ||
		o.IsTestNetwork ||
		o.IsTestVolume ||
		o.IsTestDelete
}
