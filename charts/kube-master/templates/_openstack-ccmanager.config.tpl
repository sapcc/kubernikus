{{/* vim: set filetype=gotexttmpl: */ -}}
[Global]
auth-url = {{ required "missing openstack.authURL" .Values.openstack.authURL }}
username = {{ required "missing openstack.username" .Values.openstack.username }}
password = {{ required "missing openstack.password" .Values.openstack.password }}
user-domain-name = {{ required "missing openstack.domainName" .Values.openstack.domainName }}
{{- if .Values.openstack.projectScope }}
tenant-id = {{ .Values.openstack.projectID }}
{{- end }}
{{- if .Values.openstack.region }}
region = {{ .Values.openstack.region }}
{{- end }}
[LoadBalancer]
lb-version=v2
use-octavia = {{ default "no" .Values.openstack.useOctavia }}
subnet-id= {{ required "missing openstack.lbSubnetID" .Values.openstack.lbSubnetID }}
floating-network-id= {{ required "missing openstack.lbFloatingNetworkID" .Values.openstack.lbFloatingNetworkID }}
create-monitor = yes
monitor-delay = 1m
monitor-timeout = 30s
monitor-max-retries = 3
[BlockStorage]
trust-device-path = no
[Route]
router-id = {{ required "missing openstack.routerID" .Values.openstack.routerID }}
