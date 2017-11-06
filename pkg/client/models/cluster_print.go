package models

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sapcc/kubernikus/pkg/cmd/printers"
)

func (c *Cluster) GetFormats() map[printers.PrintFormat]struct{} {
	ret := map[printers.PrintFormat]struct{}{
		printers.Table: struct{}{},
		printers.Human: struct{}{},
	}
	return ret
}

func (c *Cluster) Print(format printers.PrintFormat, options printers.PrintOptions) error {
	switch format {
	case printers.Table:
		c.printTable(options)
	case printers.Human:
		c.printHuman(options)
	default:
		return errors.Errorf("Unknown printformat models.Cluster is unable to print in format: %v", format)
	}
	return nil
}

func (c *Cluster) printHuman(options printers.PrintOptions) {
	fmt.Println("Cluster name: ", *c.Name)
	fmt.Println("Cluster state: ", (*c).Status.Kluster.State)
	fmt.Println("Cluster CIDR: ", (*c).Spec.ClusterCIDR)
	fmt.Println("Service CIDR: ", (*c).Spec.ServiceCIDR)
	fmt.Println("Cluster node pools: ", len((*c).Spec.NodePools))
	for _, pool := range (*c).Spec.NodePools {
		pool.print()
	}
	fmt.Println("Cluster node pool status: ")
	for _, pool := range (*c).Status.NodePools {
		pool.print()
	}
}

func (p *ClusterStatusNodePoolsItems0) print() {
	fmt.Print("Name: ")
	fmt.Println(*p.Name)
	fmt.Print("   Size: \t")
	fmt.Println(*p.Size)
	fmt.Print("   Running: \t")
	fmt.Println(*p.Running)
	fmt.Print("   Schedulable: \t")
	fmt.Println(*p.Schedulable)
	fmt.Print("   Healthy: \t")
	fmt.Println(*p.Healthy)
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
