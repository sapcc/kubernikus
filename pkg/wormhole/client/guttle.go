package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/databus23/guttle"
	"github.com/go-kit/log"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func New(kubeconfig, context, serverAddr, listenAddr string, logger log.Logger) (*guttle.Client, error) {

	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig file %#v: %s", kubeconfig, err)
	}
	err = api.FlattenConfig(config)
	if err != nil {
		return nil, err
	}

	if context == "" {
		context = config.CurrentContext
	}
	if context == "" {
		return nil, fmt.Errorf("no context given")
	}

	ctx, found := config.Contexts[context]
	if !found {
		return nil, fmt.Errorf("context %s not found", context)
	}

	cluster, found := config.Clusters[ctx.Cluster]
	if !found {
		return nil, fmt.Errorf("cluster not found %s", ctx.Cluster)
	}

	authInfo, found := config.AuthInfos[ctx.AuthInfo]
	if !found {
		return nil, fmt.Errorf("no auth info found for context %s", ctx.AuthInfo)
	}
	cert := authInfo.ClientCertificateData
	key := authInfo.ClientKeyData

	ca := cluster.CertificateAuthorityData

	var rootCAs *x509.CertPool
	if ca != nil {
		rootCAs = x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("failed to load any certs from %s", ca)
		}
	}
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate/key: %s", err)
	}

	if serverAddr == "" {
		url, err := url.Parse(cluster.Server)
		if err != nil {
			return nil, err
		}
		c := strings.Split(url.Hostname(), ".")
		//Add "-t" to first component of hostname
		c[0] = fmt.Sprintf("%s-wormhole", c[0])
		serverAddr = fmt.Sprintf("%s:%s", strings.Join(c, "."), "443")
	}

	opts := guttle.ClientOptions{
		Logger:     logger,
		ServerAddr: serverAddr,
		ListenAddr: listenAddr,
		Dial: func(network, address string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			conn, err := tls.DialWithDialer(dialer, network, address, &tls.Config{
				RootCAs:      rootCAs,
				Certificates: []tls.Certificate{certificate},
			})
			if err != nil {
				logger.Log(
					"msg", "failed to open connection",
					"address", address,
					"err", err)
			}
			return conn, err
		},
	}

	return guttle.NewClient(&opts), nil
}
