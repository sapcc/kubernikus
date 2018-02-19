package main

import "time"

const (
	Timeout       = 10 * time.Minute
	CheckInterval = 10 * time.Second

	TimeoutPod = 5 * time.Minute

	ClusterName               = "e2e"
	ClusterSmallNodePoolSize  = 2
	ClusterMediumNodePoolSize = 1

	NginxName  = "e2e-nginx"
	NginxImage = "circa10a/nginx-wget" //ngnix with wget
	NginxPort  = 80
	Namespace  = "default"

	WGETRetries = 12
	WGETTimeout = 10

	PVCSize      = "1Gi"
	PVCName      = "e2e-nginx-pvc"
	PVCMountPath = "/mymount"

	PathBin                 = "/usr/bin"
	KubernikusctlBinaryName = "kubernikusctl"
)
