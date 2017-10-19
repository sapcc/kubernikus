package printers

import (
	"github.com/pkg/errors"
)

type PrintOptions struct {
	WithHeaders bool
}

type Printable interface {
	GetFormats() map[string]struct{}
	Print(format string, options PrintOptions) error
}

func PrintTable(list []Printable) error {
	first := true
	for _, item := range list {
		_, ok := item.GetFormats()["table"]
		if !ok {
			return errors.Errorf("Unsupported print format table supported formats: %v, %v", item, item.GetFormats())
		}
		item.Print("table", PrintOptions{WithHeaders: first})
		first = false
	}
	return nil
}
