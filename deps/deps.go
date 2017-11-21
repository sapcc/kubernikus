//this file is only here so that `glide list` picks
//up our dependencies not used in code.
//Otherwise glide-vc deletes it
package extra_dependencies

import (
	_ "github.com/stretchr/testify/assert"
	_ "k8s.io/client-go/kubernetes/fake"
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/informer-gen"
	_ "k8s.io/code-generator/cmd/lister-gen"
)
