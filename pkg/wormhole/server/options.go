package server

import "github.com/go-kit/kit/log"

type Options struct {
	Logger log.Logger

	//Used by the controller
	KubeConfig  string
	Context     string
	ServiceCIDR string

	//Used by the tunnel
	ClientCA    string
	Certificate string
	PrivateKey  string

	ApiPort     int
}
