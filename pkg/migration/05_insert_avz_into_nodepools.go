package migration

import (
	"fmt"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/util"
)

func InsertAVZIntoNodePools(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {
	secret, err := util.KlusterSecret(clients.Kubernetes, current)
	if err != nil {
		return err
	}

	avz := ""
	switch secret.Region {
	case "ap-ae-1":
		avz = "ap-ae-1a"
	case "ap-au-1":
		avz = "ap-au-1b"
	case "ap-cn-1":
		avz = "ap-cn-1a"
	case "ap-jp-1":
		avz = "ap-jp-1a"
	case "ap-jp-2":
		avz = "ap-jp-2a"
	case "ap-sa-1":
		avz = "ap-sa-1a"
	case "eu-de-1":
		avz = "eu-de-1a"
	case "eu-de-2":
		avz = "eu-de-2a"
	case "eu-nl-1":
		avz = "eu-nl-1b"
	case "eu-ru-1":
		avz = "eu-ru-1b"
	case "la-br-1":
		avz = "la-br-1a"
	case "na-ca-1":
		avz = "na-ca-1a"
	case "na-us-1":
		avz = "na-us-1a"
	case "na-us-3":
		avz = "na-us-3a"
	case "qa-de-1":
		avz = "qa-de-1a"
	default:
		return fmt.Errorf("couldn't find default AVZ for region %s", secret.Region)
	}

	for i, pool := range current.Spec.NodePools {
		if pool.AvailabilityZone == "" {
			current.Spec.NodePools[i].AvailabilityZone = avz
		}
	}

	return err
}
