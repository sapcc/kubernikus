package main

import (
	goflag "flag"
	"log"
	"os"
	"os/signal"

	"github.com/spf13/pflag"

	"github.com/golang/glog"
	"sync"
	"syscall"
	"testing"
)

var options E2ETestSuiteOptions

func init() {
	pflag.StringVar(&options.ConfigFile, "configFile", "test/e2e/e2e_config.yaml", "Path to configuration file")
	pflag.BoolVar(&options.IsTestCreate, "create", false, "Create a new cluster")
	pflag.BoolVar(&options.IsTestNetwork, "network", false, "Run network tests")
	pflag.BoolVar(&options.IsTestNetwork, "volume", false, "Run volume tests")
	pflag.BoolVar(&options.IsTestDelete, "delete", false, "Delete the cluster")
	pflag.BoolVar(&options.IsTestAll, "all", false, "The whole show. Test everything")
	pflag.BoolVar(&options.IsTestAPI, "api", false, "Test API")
	pflag.BoolVar(&options.IsTestSmoke, "smoke", false, "Run smoke test")
}

func main() {
	log.SetOutput(os.Stdout)

	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()

	sigs := make(chan os.Signal, 1)
	stop := make(chan bool)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM) // Push signals into channel

	wg := &sync.WaitGroup{} // Goroutines can add themselves to this to be waited on

	testSuite := NewE2ETestSuite(&testing.T{}, options)
	if testSuite == nil {
		glog.Fatal("Couldn't create e2e test suite. Aborting")
	}

	go testSuite.Run(wg, sigs, stop)

	<-sigs // Wait for signals (this hangs until a signal arrives)
	log.Println("The whole system goes down...")
	close(stop) // Tell goroutines to stop themselves
	wg.Wait()   // Wait for all to be stopped
}
