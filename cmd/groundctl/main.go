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

		go ground.New(ground.Options{}).Run(1, stop, wg)

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
	goflag.CommandLine.Parse([]string{}) //https://github.com/kubernetes/kubernetes/issues/17162

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
		glog.V(2).Info("%v", err)
	} else {
		glog.V(2).Infof(viper.ConfigFileUsed())
	}
}
