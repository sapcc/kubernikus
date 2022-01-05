package models

import (
	"sort"
)

type flavorSorter struct {
	flavors []Flavor
}

func SortFlavors(flavors []Flavor) {
	sort.Sort(&flavorSorter{flavors: flavors})
}

//Part of sort.Interface
func (fs *flavorSorter) Len() int {
	return len(fs.flavors)
}

func (fs *flavorSorter) Swap(i, j int) {
	fs.flavors[i], fs.flavors[j] = fs.flavors[j], fs.flavors[i]
}

func (fs *flavorSorter) Less(i, j int) bool {
	if fs.flavors[i].RAM == fs.flavors[j].RAM {
		return fs.flavors[i].Vcpus < fs.flavors[j].Vcpus
	}
	return fs.flavors[i].RAM < fs.flavors[j].RAM
}
