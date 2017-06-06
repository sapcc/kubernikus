package main

import (
	goflag "flag"

	flag "github.com/spf13/pflag"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RootCmd = &cobra.Command{
	Use: "groundctl",
}

var satelliteName string
var configFile string

func main() {
	RootCmd.Execute()
}

func init() {
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is /etc/kubernikus/groundctl.yaml)")
}

func initConfig() {
	if configFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(configFile)
	}

	viper.SetConfigName("groundctl")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath("/etc/kubernikus")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		glog.Fatalf("%v", err)
	} else {
		glog.V(2).Infof(viper.ConfigFileUsed())
	}
}
