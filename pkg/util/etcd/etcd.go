package etcd

import (
	"fmt"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

const (
	EtcdBackupStorageContainer = "etcd-backup-%s-%s"
)

func DefaultStorageContainer(kluster *v1.Kluster) string {
	return fmt.Sprintf(EtcdBackupStorageContainer, kluster.GetName(), kluster.GetUID())
}
