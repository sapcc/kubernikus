package handlers

import (
	"sort"

	"github.com/Masterminds/semver"
	"github.com/go-openapi/runtime/middleware"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	"github.com/sapcc/kubernikus/pkg/version"
)

func NewInfo(rt *api.Runtime) operations.InfoHandler {
	i := info{Runtime: rt}
	if rt.Images != nil {
		for version, metadata := range rt.Images.Versions {
			if metadata.Default {
				i.defaultVersion = version
			}
			if metadata.Supported {
				i.supportedVersions = append(i.supportedVersions, version)
			}
			i.availableVersions = append(i.availableVersions, version)
		}
		if len(i.supportedVersions) > 1 {
			i.supportedVersions = sortVersions(i.supportedVersions)
		}
		if len(i.availableVersions) > 1 {
			i.availableVersions = sortVersions(i.availableVersions)
		}
	}
	return &i
}

type info struct {
	*api.Runtime
	supportedVersions []string
	availableVersions []string
	defaultVersion    string
}

func (d *info) Handle(params operations.InfoParams) middleware.Responder {

	info := &models.Info{
		GitVersion:               version.GitCommit,
		SupportedClusterVersions: d.supportedVersions,
		AvailableClusterVersions: d.availableVersions,
		DefaultClusterVersion:    d.defaultVersion,
	}
	return operations.NewInfoOK().WithPayload(info)
}

func sortVersions(in []string) []string {
	vs := make([]*semver.Version, len(in))
	for i, r := range in {
		v, err := semver.NewVersion(r)
		if err != nil {
			return in
		}
		vs[i] = v
	}
	collection := semver.Collection(vs)
	sort.Sort(collection)
	out := make([]string, 0, len(collection))
	for _, v := range collection {
		out = append(out, v.String())
	}
	return out
}
