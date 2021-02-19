[Global]
auth-url = {{ required "missing openstack.authURL" .Values.openstack.authURL }}
domain-name = {{ required "missing openstack.domainName" .Values.openstack.domainName }}
tenant-id = {{ required "missing openstack.projectID" .Values.openstack.projectID }}
username = {{ required "missing openstack.username" .Values.openstack.username }}
password = {{ required "missing openstack.password" .Values.openstack.password }}
region = {{ required "missing openstack.region" .Values.openstack.region }}

[BlockStorage]
