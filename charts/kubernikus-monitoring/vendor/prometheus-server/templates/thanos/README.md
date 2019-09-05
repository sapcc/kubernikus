Thanos
------

[Thanos](https://github.com/improbable-eng/thanos) ready for SAP Converged Cloud.

Enabling Thanos will install the following components:
- Thanos compactor
- Thanos query
- Thanos store
- OpenstackSeed for Swift service user and container
- Thanos sidecar for Prometheus server

## Configuration

Minimal configuration for Thanos@SAP Converged Cloud:

```
# How long metrics are retained. 
# For Thanos you want something like 1y.
retentionTime: <retention>

# In memory. Persist to Swift.
persistence:
  enabled: false

thanos:
  enabled: true
  
  # Create an OpenStack user in the given scope with required permissions and initialize container in Swift.
  swiftStorageConfig:
    authURL:            https://<keystone>/v3
    userName:           <userName>
    userDomainName:     <userDomainName>
    password:           <password>
    domainName:         <domainName>
    projectName:        <projectName>
    projectDomainName:  <projectDomainName>
    regionName:         <regionName>
    containerName:      <swiftContainerName>
```

An existing OpenStack user and Swift container can be used by disabled the OpenStack seed:
```
thanos:
  seed:
    enabled: false
  
  swiftStorageConfig:
    ...
```

**Note**: The user must have permissions to read from/write to the Swift container in the given scope.  

## Troubleshooting

### Creation of the OpenStack user

Thanos, as configured in this chart, uses OpenStack Swift as the backend.
Thus an OpenStack service user with sufficient access is required.
Luckily it is created without manual intervention using a Kubernetes operator, the [Openstack Seeder](https://github.com/sapcc/kubernetes-operators/tree/master/openstack-seeder).
Its configuration is deployed via [OpenStackSeed](thanos-seed.yaml) for the Thanos service user.  
However, creation of the OpenStack service user for Thanos might take longer than the Thanos components need to be up & running.
In which case they might end up in `Error` state.
This needs manual intervention: 
1. Ensure the OpenStack service user and container was successfully created.
2. Delete the Pod of the Prometheus StatefulSet and of the related Thanos components.
