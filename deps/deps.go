//this file is only here so that `glide list` picks
//up our dependencies not used in code.
//Otherwise glide-vc deletes it
package extra_dependencies

import (
	_ "github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/quotasets"
	_ "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/quotasets"
	_ "github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	_ "github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/mock"
	_ "github.com/stretchr/testify/require"
	_ "github.com/stretchr/testify/suite"
	_ "k8s.io/client-go/kubernetes/fake"
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "k8s.io/code-generator/cmd/informer-gen"
	_ "k8s.io/code-generator/cmd/lister-gen"
)
