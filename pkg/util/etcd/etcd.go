package etcd

import (
	"fmt"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

const (
	BackupStorageContainerBase = "kubernikus-etcd-backup"
)

func DefaultStorageContainer(kluster *v1.Kluster) string {
	return fmt.Sprintf("%s-%s-%s", BackupStorageContainerBase, kluster.Spec.Name, kluster.GetUID())
}
