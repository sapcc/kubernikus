package migration

import (
	"k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

//Init is the first migration that only sets the SpecVersion to 1
func Init(rawKluster []byte, current *v1.Kluster, client kubernetes.Interface) (err error) {
	return nil
}
