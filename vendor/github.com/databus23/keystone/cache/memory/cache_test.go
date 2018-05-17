package memory

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	c := New(5 * time.Second)
	c.Set("test", "blafasel", 1*time.Second)

	var value string
	if ok := c.Get("test", &value); !ok || value != "blafasel" {
		t.Fatalf("Expected %q, got %q", "blafasel", value)
	}

}
