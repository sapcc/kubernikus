package migration

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

func NoOp(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {
	return nil
}
