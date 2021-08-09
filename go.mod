module github.com/sapcc/kubernikus

go 1.16

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/goutils v1.1.0
	github.com/Masterminds/semver v1.3.1
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/ajeddeloh/yaml v0.0.0-20160722214022-1072abfea311 // indirect
	github.com/aokoli/goutils v1.0.1
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a
	github.com/coreos/container-linux-config-transpiler v0.4.2
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/ignition v0.17.2
	github.com/danieljoos/wincred v1.0.1 // indirect
	github.com/databus23/goslo.policy v0.0.0-20170317131957-3ae74dd07ebf
	github.com/databus23/guttle v0.0.0-20180831113238-290306d52eb2
	github.com/databus23/keystone v0.0.0-20180111110916-350fd0e663cd
	github.com/databus23/requestutil v0.0.0-20160108082528-5ff8e981f38f
	github.com/ghodss/yaml v1.0.0
	github.com/go-kit/kit v0.10.0
	github.com/go-openapi/errors v0.19.2
	github.com/go-openapi/loads v0.19.2
	github.com/go-openapi/runtime v0.19.0
	github.com/go-openapi/spec v0.19.2
	github.com/go-openapi/strfmt v0.19.0
	github.com/go-openapi/swag v0.19.5
	github.com/go-openapi/validate v0.19.2
	github.com/go-stack/stack v1.8.0
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/godbus/dbus v0.0.0-20151105175453-c7fdd8b5cd55 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/gophercloud/gophercloud v0.1.0
	github.com/hashicorp/yamux v0.0.0-20180826203732-cc6d2ea263b2 // indirect
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.5
	github.com/joeshaw/envdecode v0.0.0-20170502020559-6326cbed175e
	github.com/justinas/alice v0.0.0-20171023064455-03f45bd4b7da
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/oklog/run v1.0.1-0.20180308005104-6934b124db28
	github.com/pkg/errors v0.8.1
	github.com/pmylund/go-cache v2.1.0+incompatible // indirect
	github.com/prometheus/client_golang v1.3.0
	github.com/rs/cors v0.0.0-20170801073201-eabcc6af4bbe
	github.com/satori/go.uuid v1.2.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/tredoe/osutil v0.0.0-20161130133508-7d3ee1afa71c
	github.com/vincent-petithory/dataurl v0.0.0-20160330182126-9a301d65acbb // indirect
	github.com/zalando/go-keyring v0.0.0-20180221093347-6d81c293b3fb
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	golang.org/x/sys v0.0.0-20191220142924-d4481acd189f
	google.golang.org/grpc v1.26.0
	gopkg.in/yaml.v2 v2.2.8
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	k8s.io/api v0.16.15
	k8s.io/apiextensions-apiserver v0.0.0-20200831054406-27625ab65fd7
	k8s.io/apimachinery v0.16.15
	k8s.io/client-go v0.16.15
	k8s.io/cluster-bootstrap v0.0.0-20190802024125-9150a5ba960c
	k8s.io/helm v2.11.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20190801114015-581e00157fb1
)

replace k8s.io/klog => github.com/simonpasquier/klog-gokit v0.3.1-0.20210409073344-020c8028ac9e
