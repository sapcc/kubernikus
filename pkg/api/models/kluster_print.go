package models

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/sapcc/kubernikus/pkg/cmd/printers"
)

func (k Kluster) GetFormats() map[printers.PrintFormat]struct{} {
	ret := map[printers.PrintFormat]struct{}{
		printers.Table: {},
		printers.Human: {},
	}
	return ret
}

func (k Kluster) Print(format printers.PrintFormat, options printers.PrintOptions) error {
	switch format {
	case printers.Table:
		k.printTable(options)
	case printers.Human:
		k.printHuman(options)
	default:
		return errors.Errorf("Unknown printformat models.Cluster is unable to print in format: %v", format)
	}
	return nil
}

func (k Kluster) printHuman(options printers.PrintOptions) {
	fmt.Println("Cluster name: ", k.Name)
	fmt.Println("Cluster state: ", k.Status.Phase)
	fmt.Println("Cluster CIDR: ", k.Spec.ClusterCIDR)
	fmt.Println("Service CIDR: ", k.Spec.ServiceCIDR)
	fmt.Println("Cluster node pools: ", len(k.Spec.NodePools))
	for _, pool := range k.Spec.NodePools {
		pool.printHuman(options)
	}
	fmt.Println("Cluster node pool status: ")
	for _, pool := range k.Status.NodePools {
		pool.printHuman(options)
	}
}

func (k *Kluster) printTable(options printers.PrintOptions) {
	if options.WithHeaders {
		fmt.Print("NAME")
		fmt.Print("\t")
		fmt.Print("STATUS")
	}
	fmt.Print(k.Name)
	fmt.Print("\t")
	fmt.Print(k.Status.Phase)
}

func (p NodePool) GetFormats() map[printers.PrintFormat]struct{} {
	ret := map[printers.PrintFormat]struct{}{
		printers.Table: {},
		printers.Human: {},
	}
	return ret
}

func (p NodePool) Print(format printers.PrintFormat, options printers.PrintOptions) error {
	switch format {
	case printers.Human:
		p.printHuman(options)
	case printers.Table:
		p.printTable(options)
	default:
		return errors.Errorf("Unknown printformat models.Cluster is unable to print in format: %v", format)
	}
	return nil
}

func (p NodePool) printHuman(options printers.PrintOptions) {
	fmt.Print("Name: ")
	fmt.Println(p.Name)
	fmt.Print("   Flavor: \t")
	fmt.Println(p.Flavor)
	fmt.Print("   Image:  \t")
	fmt.Println(p.Image)
	fmt.Print("   Size:   \t")
	fmt.Println(p.Size)
}

func (p NodePool) printTable(options printers.PrintOptions) {
	if options.WithHeaders {
		fmt.Print("NAME")
		fmt.Print("\t")
		fmt.Print("FLAVOR")
		fmt.Print("\t")
		fmt.Print("IMAGE")
		fmt.Print("\t")
		fmt.Println("SIZE")
	}
	fmt.Print(p.Name)
	fmt.Print("\t")
	fmt.Print(p.Flavor)
	fmt.Print("\t")
	fmt.Print(p.Image)
	fmt.Print("\t")
	fmt.Println(p.Size)
}

func (p NodePoolInfo) printHuman(options printers.PrintOptions) {
	fmt.Print("Name: ")
	fmt.Println(p.Name)
	fmt.Print("   Size: \t")
	fmt.Println(p.Size)
	fmt.Print("   Running: \t")
	fmt.Println(p.Running)
	fmt.Print("   Schedulable: \t")
	fmt.Println(p.Schedulable)
	fmt.Print("   Healthy: \t")
	fmt.Println(p.Healthy)
}
