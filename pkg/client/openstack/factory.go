package openstack

import (
	"fmt"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"

	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack/admin"
	openstack_kluster "github.com/sapcc/kubernikus/pkg/client/openstack/kluster"
	openstack_project "github.com/sapcc/kubernikus/pkg/client/openstack/project"
	utillog "github.com/sapcc/kubernikus/pkg/util/log"
)

type SharedOpenstackClientFactory interface {
	KlusterClientFor(*kubernikus_v1.Kluster) (openstack_kluster.KlusterClient, error)
	ProjectClientFor(authOptions *tokens.AuthOptions) (openstack_project.ProjectClient, error)
	ProjectAdminClientFor(string) (openstack_project.ProjectClient, error)
	ProviderClientFor(authOptions *tokens.AuthOptions, logger log.Logger) (*gophercloud.ProviderClient, error)
	ProviderClientForKluster(kluster *kubernikus_v1.Kluster, logger log.Logger) (*gophercloud.ProviderClient, error)
	AdminClient() (admin.AdminClient, error)
}

type factory struct {
	klusterClients sync.Map
	projectClients sync.Map
	adminClient    admin.AdminClient

	secrets          core_v1.SecretInterface
	klusters         cache.SharedIndexInformer
	adminAuthOptions *tokens.AuthOptions
	logger           log.Logger
}

func NewSharedOpenstackClientFactory(secrets core_v1.SecretInterface, klusters cache.SharedIndexInformer, adminAuthOptions *tokens.AuthOptions, logger log.Logger) SharedOpenstackClientFactory {
	factory := &factory{
		secrets:          secrets,
		klusters:         klusters,
		adminAuthOptions: adminAuthOptions,
		logger:           logger,
	}

	if klusters != nil {
		klusters.AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				if kluster, ok := obj.(*kubernikus_v1.Kluster); ok {
					factory.logger.Log(
						"msg", "deleting shared openstack client",
						"kluster", kluster.Name,
						"v", 5)
					factory.klusterClients.Delete(kluster.GetUID())
				}
			},
		})
	}

	return factory
}

func (f *factory) AdminClient() (admin.AdminClient, error) {
	if f.adminClient != nil {
		return f.adminClient, nil
	}

	identity, compute, network, err := f.serviceClientsFor(f.adminAuthOptions, f.logger)
	if err != nil {
		return nil, err
	}

	var client admin.AdminClient
	client = admin.NewAdminClient(network, compute, identity)
	client = admin.LoggingClient{Client: client, Logger: f.logger}

	f.adminClient = client

	return f.adminClient, nil
}

func (f *factory) KlusterClientFor(kluster *kubernikus_v1.Kluster) (openstack_kluster.KlusterClient, error) {
	if obj, found := f.klusterClients.Load(kluster.GetUID()); found {
		return obj.(openstack_kluster.KlusterClient), nil
	}

	authOptions, err := f.authOptionsForKluster(kluster)
	if err != nil {
		return nil, err
	}

	identity, compute, network, err := f.serviceClientsFor(authOptions, f.logger)
	if err != nil {
		return nil, err
	}

	var client openstack_kluster.KlusterClient
	client = openstack_kluster.NewKlusterClient(network, compute, identity, kluster)
	client = &openstack_kluster.LoggingClient{Client: client, Logger: log.With(f.logger, "kluster", kluster.GetName(), "project", kluster.Account())}

	f.klusterClients.Store(kluster.GetUID(), client)

	return client, nil
}

func (f *factory) ProjectClientFor(authOptions *tokens.AuthOptions) (openstack_project.ProjectClient, error) {
	if authOptions.Scope.ProjectID == "" {
		return nil, fmt.Errorf("AuthOptions must be scoped to a projectID")
	}
	return f.projectClient(authOptions.Scope.ProjectID, authOptions)
}

func (f *factory) ProjectAdminClientFor(projectID string) (openstack_project.ProjectClient, error) {
	return f.projectClient(projectID, f.adminAuthOptions)
}

func (f *factory) projectClient(projectID string, authOptions *tokens.AuthOptions) (openstack_project.ProjectClient, error) {
	if obj, found := f.projectClients.Load(projectID); found {
		return obj.(openstack_project.ProjectClient), nil
	}

	identity, compute, network, err := f.serviceClientsFor(authOptions, f.logger)
	if err != nil {
		return nil, err
	}

	var client openstack_project.ProjectClient
	client = openstack_project.NewProjectClient(projectID, network, compute, identity)
	client = &openstack_project.LoggingClient{Client: client, Logger: log.With(f.logger, "project_id", projectID)}

	f.projectClients.Store(projectID, client)
	return client, nil
}

func (f *factory) authOptionsForKluster(kluster *kubernikus_v1.Kluster) (*tokens.AuthOptions, error) {
	secret, err := f.secrets.Get(kluster.Name, meta_v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Couldn't retrieve secret %s/%s: %v", kluster.GetNamespace(), kluster.Name, err)
	}

	authOptions := &tokens.AuthOptions{
		IdentityEndpoint: string(secret.Data["openstack-auth-url"]),
		Username:         string(secret.Data["openstack-username"]),
		Password:         string(secret.Data["openstack-password"]),
		DomainName:       string(secret.Data["openstack-domain-name"]),
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectID: string(secret.Data["openstack-project-id"]),
		},
	}

	f.logger.Log(
		"msg", "using authOptions from secret",
		"identity_endpoint", authOptions.IdentityEndpoint,
		"username", authOptions.Username,
		"domain_name", authOptions.DomainName,
		"project_id", authOptions.Scope.ProjectID,
		"v", 5)

	return authOptions, nil
}

func (f *factory) ProviderClientForKluster(kluster *kubernikus_v1.Kluster, logger log.Logger) (*gophercloud.ProviderClient, error) {
	authOptions, err := f.authOptionsForKluster(kluster)
	if err != nil {
		return nil, err
	}
	return f.ProviderClientFor(authOptions, logger)
}

func (f *factory) ProviderClientFor(authOptions *tokens.AuthOptions, logger log.Logger) (*gophercloud.ProviderClient, error) {
	provider, err := utillog.NewLoggingProviderClient(authOptions.IdentityEndpoint, logger)
	if err != nil {
		return nil, err
	}

	provider.UseTokenLock()

	err = openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	return provider, nil
}

func (f *factory) serviceClientsFor(authOptions *tokens.AuthOptions, logger log.Logger) (*gophercloud.ServiceClient, *gophercloud.ServiceClient, *gophercloud.ServiceClient, error) {
	providerClient, err := f.ProviderClientFor(authOptions, logger)
	if err != nil {
		return nil, nil, nil, err
	}

	identity, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, nil, nil, err
	}

	compute, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{})
	compute.Microversion = "2.25" // 2.25 is the maximum in mitaka. we need at least 2.15 to create `soft-affinity` server groups
	if err != nil {
		return nil, nil, nil, err
	}

	network, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, nil, nil, err
	}

	return identity, compute, network, nil
}
