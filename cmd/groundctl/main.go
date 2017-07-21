package main

import (
	goflag "flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sapcc/kubernikus/pkg/controller/ground"
	flag "github.com/spf13/pflag"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RootCmd = &cobra.Command{
	Use: "groundctl",
	RunE: func(cmd *cobra.Command, args []string) error {
		sigs := make(chan os.Signal, 1)
		stop := make(chan struct{})
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM) // Push signals into channel

		wg := &sync.WaitGroup{} // Goroutines can add themselves to this to be waited on

		options := ground.Options{
			ChartDirectory:    viper.GetString("chart-directory"),
			AuthURL:           viper.GetString("auth-url"),
			AuthDomain:        viper.GetString("auth-domain"),
			AuthUsername:      viper.GetString("auth-username"),
			AuthPassword:      viper.GetString("auth-password"),
			AuthProject:       viper.GetString("auth-project"),
			AuthProjectDomain: viper.GetString("auth-project-domain"),
		}

		go ground.New(options).Run(1, stop, wg)

		<-sigs // Wait for signals (this hangs until a signal arrives)
		glog.Info("Shutting down...")

		close(stop) // Tell goroutines to stop themselves
		wg.Wait()   // Wait for all to be stopped
		return nil
	},
}

var satelliteName string
var configFile string

func main() {
	RootCmd.Execute()
}

func init() {
	// parse the CLI flags
	if f := goflag.Lookup("logtostderr"); f != nil {
		f.Value.Set("true") // log to stderr by default
	}
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	//goflag.CommandLine.Parse([]string{}) //https://github.com/kubernetes/kubernetes/issues/17162

	cobra.OnInitialize(initConfig)
	viper.AutomaticEnv()
	viper.SetEnvPrefix("KUBERNIKUS")

	RootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is /etc/kubernikus/groundctl.yaml)")
	RootCmd.Flags().String("chart-directory", "charts/", "Directory containing the kubernikus related charts")
	viper.BindPFlag("chart-directory", RootCmd.Flags().Lookup("chart-directory"))

	RootCmd.Flags().String("auth-url", "http://keystone.monsoon3:5000/v3", "Openstack keystone url")
	viper.BindPFlag("auth-url", RootCmd.Flags().Lookup("auth-url"))

	RootCmd.Flags().String("auth-username", "kubernikus", "Service user for kubernikus")
	viper.BindPFlag("auth-username", RootCmd.Flags().Lookup("auth-username"))

	RootCmd.Flags().String("auth-password", "", "Service user password")
	viper.BindPFlag("auth-password", RootCmd.Flags().Lookup("auth-password"))

	RootCmd.Flags().String("auth-domain", "Default", "Service user domain")
	viper.BindPFlag("auth-domain", RootCmd.Flags().Lookup("auth-domain"))

	RootCmd.Flags().String("auth-project", "", "Scope service user to this project")
	viper.BindPFlag("auth-project", RootCmd.Flags().Lookup("auth-project"))

	RootCmd.Flags().String("auth-project-domain", "", "Domain of the project")
	viper.BindPFlag("auth-project-domain", RootCmd.Flags().Lookup("auth-project-domain"))
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
		glog.V(2).Infof("Not using any config file: %s", err)
	} else {
		glog.V(2).Infof("Loaded config %s", viper.ConfigFileUsed())
	}
}
