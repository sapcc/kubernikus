package models

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sapcc/kubernikus/pkg/cmd/printers"
)

func (c *Cluster) GetFormats() map[string]struct{} {
	ret := map[string]struct{}{
		"table": struct{}{},
		"human": struct{}{},
	}
	return ret
}

func (c *Cluster) Print(format string, options printers.PrintOptions) error {
	switch format {
	case "table":
		c.printTable(options)
	case "human":
		c.printHuman(options)
	default:
		return errors.Errorf("Unknown printformat models.Cluster is unable to print in format: %v", format)
	}
	return nil
}

func (c *Cluster) printHuman(options printers.PrintOptions) {
	fmt.Println("Cluster name: ", *c.Name)
	fmt.Println("Cluster state: ", (*c).Status.Kluster.State)
	if (*c).Spec != nil {
		fmt.Println("Cluster node pools: ", len((*c).Spec.NodePools))
		for _, pool := range (*c).Spec.NodePools {
			pool.print()
		}
	}
}

func (p *ClusterSpecNodePoolsItems0) print() {
	fmt.Print("Name: ")
	fmt.Println(*p.Name)
	fmt.Print("   Flavor: \t")
	fmt.Println(*p.Flavor)
	fmt.Print("   Image:  \t")
	fmt.Println(p.Image)
	fmt.Print("   Size:   \t")
	fmt.Println(*p.Size)
}

func (c *Cluster) printTable(options printers.PrintOptions) {
	if options.WithHeaders {
		fmt.Print("NAME")
		fmt.Print("\t")
		fmt.Print("STATUS")
		fmt.Print("\t")
		fmt.Println("MESSAGE")
	}
	fmt.Print(*c.Name)
	fmt.Print("\t")
	fmt.Print((*c).Status.Kluster.State)
	fmt.Print("\t")
	fmt.Println((*c).Status.Kluster.Message)
}
