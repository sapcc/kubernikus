package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlavorSorter(t *testing.T) {

	m1small := Flavor{ID: "m1.small", RAM: 2046, Vcpus: 2}
	m1xsmall := Flavor{ID: "m1.xsmall", RAM: 4096, Vcpus: 2}
	m1medium := Flavor{ID: "m1.medium", RAM: 4096, Vcpus: 4}
	m1xmedium := Flavor{ID: "m1.xmedium", RAM: 8192, Vcpus: 4}

	input := []Flavor{m1xmedium, m1medium, m1small, m1xsmall}

	SortFlavors(input)

	expected := []Flavor{m1small, m1xsmall, m1medium, m1xmedium}

	assert.Equal(t, expected, input)

}
