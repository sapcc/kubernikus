package spec

import "testing"

func TestDefaulString(t *testing.T) {
	value, err := DefaultString("KlusterSpec", "serviceCIDR")
	if err != nil {
		t.Fatal("returned an error ", err)
	}
	if value == "" {
		t.Fatal("did not return a non-empty value")
	}
}
