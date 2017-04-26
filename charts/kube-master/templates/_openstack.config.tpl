{{/* vim: set filetype=gotexttmpl: */ -}}
[Global]
auth-url = {{ .authUrl }}
username = {{ .username }} 
password = {{ .password }} 
domain-name = {{ .domainName }} 
tenant-name = {{ .prohectName }} 
region = {{ .region }} 
[LoadBalancer]
lb-version=v2
subnet-id= {{ .lbSubnetID }}
create-monitor = yes
monitor-delay = 1m
monitor-timeout = 30s
monitor-max-retries = 3
[BlockStorage]
trust-device-path = no
[Route]
router-id = {{ .routerID }}
