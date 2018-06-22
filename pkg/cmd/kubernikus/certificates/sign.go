package certificates

import (
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"

	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/cmd"
	"github.com/sapcc/kubernikus/pkg/util"
	logutil "github.com/sapcc/kubernikus/pkg/util/log"
)

func NewSignCommand() *cobra.Command {
	o := NewSignOptions()

	c := &cobra.Command{
		Use:   "sign KLUSTER",
		Short: "Sign",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c))
		},
	}
	o.BindFlags(c.Flags())
	return c
}

type SignOptions struct {
	Name         string
	KubeConfig   string
	Namespace    string
	CN           string
	CA           string
	Organization string
	ApiURL       string
	LogLevel     int
}

func NewSignOptions() *SignOptions {
	return &SignOptions{
		Namespace:    "kubernikus",
		CA:           "apiserver-clients-ca",
		CN:           os.Getenv("USER"),
		Organization: "system:masters",
	}
}

func (o *SignOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Path to the kubeconfig file to use to talk to the Kubernetes apiserver. If unset, try the environment variable KUBECONFIG, as well as in-cluster configuration")
	flags.StringVar(&o.Namespace, "namespace", o.Namespace, "Namespace where the kluster is located")
	flags.StringVar(&o.CN, "cn", o.CN, "Common name in the certificate")
	flags.StringVar(&o.Organization, "organization", o.Organization, "Common name in the certificate")
	flags.StringVar(&o.ApiURL, "api-url", o.ApiURL, "URL for the apiserver")
	flags.IntVar(&o.LogLevel, "v", 0, "log level")
}

func (o *SignOptions) Validate(c *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("you must specify the kluster resource name")
	}

	if o.CN == "" {
		return errors.New("You must specify a common name")
	}
	if o.ApiURL == "" {
		return errors.New("You must specify an apiserver URL")
	}

	return nil
}

func (o *SignOptions) Complete(args []string) error {
	o.Name = args[0]
	return nil
}

func (o *SignOptions) Run(c *cobra.Command) error {
	logger := logutil.NewLogger(o.LogLevel)
	client, err := kubernetes.NewClient(o.KubeConfig, "", logger)
	if err != nil {
		return err
	}
	secret, err := client.CoreV1().Secrets(o.Namespace).Get(o.Name, v1.GetOptions{})
	if err != nil {
		return err
	}

	clientCAKey, ok := secret.Data[fmt.Sprintf("%s-key.pem", o.CA)]
	if !ok {
		return fmt.Errorf("CA %s not found in kluster secrets", o.CA)
	}
	clientCACert, ok := secret.Data[fmt.Sprintf("%s.pem", o.CA)]
	if !ok {
		return fmt.Errorf("Key for CA %s not found in kluster secrets", o.CA)
	}

	serverCACert, ok := secret.Data["tls-ca.pem"]
	if !ok {
		return fmt.Errorf("Server CA certificate not found")
	}

	bundle, err := util.NewBundle(clientCAKey, clientCACert)
	if err != nil {
		return err
	}
	cert := bundle.Sign(util.Config{
		Sign:         o.CN,
		Organization: []string{o.Organization},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})

	config := kubernetes.NewClientConfigV1(
		o.Name,
		os.Getenv("USER"),
		o.ApiURL,
		certutil.EncodePrivateKeyPEM(cert.PrivateKey),
		certutil.EncodeCertPEM(cert.Certificate),
		serverCACert,
	)
	kubeconfig, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	fmt.Println(string(kubeconfig))

	return nil
}
