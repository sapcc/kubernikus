{{/* vim: set filetype=gotexttmpl: */ -}}
[Global]
auth-url = {{ required "missing openstack.authURL" .Values.openstack.authURL }}
username = {{ required "missing openstack.username" .Values.openstack.username }} 
password = {{ required "missing openstack.password" .Values.openstack.password }} 
domain-name = {{ required "missing openstack.domainName" .Values.openstack.domainName }} 
tenant-name = {{ required "missing openstack.projectName" .Values.openstack.projectName }} 
region = {{ required "missing openstack.region" .Values.openstack.region }} 
[LoadBalancer]
lb-version=v2
subnet-id= {{ required "missing openstack.loadBalanderSubnetID" .Values.openstack.loadBalancerSubnetID }}
create-monitor = yes
monitor-delay = 1m
monitor-timeout = 30s
monitor-max-retries = 3
[BlockStorage]
trust-device-path = no
[Route]
router-id = {{ required "missing openstack.routerID" .Values.openstack.routerID }}
