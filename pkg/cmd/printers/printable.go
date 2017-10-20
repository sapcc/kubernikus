package printers

import (
	"github.com/pkg/errors"
)

type PrintFormat int

const (
	Human PrintFormat = iota
	Table
)

type PrintOptions struct {
	WithHeaders bool
}

type Printable interface {
	GetFormats() map[PrintFormat]struct{}
	Print(format PrintFormat, options PrintOptions) error
}

func PrintTable(list []Printable) error {
	first := true
	for _, item := range list {
		_, ok := item.GetFormats()[Table]
		if !ok {
			return errors.Errorf("Unsupported print format table supported formats: %v, %v", item, item.GetFormats())
		}
		item.Print(Table, PrintOptions{WithHeaders: first})
		first = false
	}
	return nil
}
