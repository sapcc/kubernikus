package migration

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

//Init is the first migration that only sets the SpecVersion to 1
func Init(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {
	return nil
}
