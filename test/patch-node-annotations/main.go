package main

import (
	"flag"
	"fmt"
	"log"

	kitlog "github.com/go-kit/log"

	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/util"
)

func main() {

	kubeconfig := flag.String("kubeconfig", "", "")
	context := flag.String("context", "", "")
	node := flag.String("node", "", "")
	key := flag.String("key", "", "")
	val := flag.String("val", "", "")
	flag.Parse()

	client, err := kubernetes.NewClient(*kubeconfig, *context, kitlog.NewNopLogger())
	if err != nil {
		log.Fatal(err)
	}

	if *val == "" {
		fmt.Printf("Removing annotation %s from node %s\n", *key, *node)
		err := util.RemoveNodeAnnotation(*node, *key, client)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("Adding/Updating annoation %s=%s on node %s\n", *key, *val, *node)
		err := util.AddNodeAnnotation(*node, *key, *val, client)
		if err != nil {
			log.Fatal(err)
		}
	}
}
