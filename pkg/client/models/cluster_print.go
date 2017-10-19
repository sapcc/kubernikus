package models

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sapcc/kubernikus/pkg/cmd/printers"
)

func (c *Cluster) GetFormats() map[string]struct{} {
	ret := map[string]struct{}{
		"table": struct{}{},
	}
	return ret
}

func (c *Cluster) Print(format string, options printers.PrintOptions) error {
	switch format {
	case "table":
		c.PrintTable(options)
	default:
		return errors.Errorf("Unknown printformat models.Cluster is unable to print in format: %v", format)
	}
	return nil
}

func (c *Cluster) PrintTable(options printers.PrintOptions) {
	if options.WithHeaders {
		fmt.Print("NAME")
		fmt.Print("\t")
		fmt.Println("STATUS")
	}
	fmt.Print(c.Name)
	fmt.Print("\t")
	fmt.Println(c.Status)
}
