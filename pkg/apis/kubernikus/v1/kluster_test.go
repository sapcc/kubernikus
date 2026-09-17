package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCARotation(t *testing.T) {
	k := &Kluster{ObjectMeta: metav1.ObjectMeta{
		Annotations: map[string]string{RotateCAAnnotation: "true"},
	}}
	assert.True(t, k.CARotation())

	k.Annotations = nil
	assert.False(t, k.CARotation())

	k.Annotations = map[string]string{RotateCAAnnotation: "false"}
	assert.False(t, k.CARotation())
}
