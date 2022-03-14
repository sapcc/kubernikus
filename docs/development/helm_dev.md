---
title: Developing Helm Charts
---

## Helm value generation

When developing on the Helm charts you can generate the values file (including certs) via
```kubernikus helm --with --many --options > values.yaml```  

You will need to provide all of the options which are:  
``` --api https://k-eu-nl-1.admin.cloud.sap``` the region of kubernikus you are interacting with  
``` --auth-url https://identity-3.eu-nl-1.cloud.sap/v3/``` the identity instance of that region (_v3_ is important)  
``` --auth-project marian``` the name of the project you are interacting with  
``` --auth-domain kubernikus``` the domain of the user you want to use  
``` --project-id 513be7d11ee64ed7826f5c7c4cfdf10c```  
``` --auth-password supersecret``` the password of the user  
``` --auth-project-domain monsoon3``` the domain of the project  
``` --auth-username techuser1``` the user with which to interact  

### Changing values.yaml
After the fact some values have to be added to the ```values.yaml```:
```
api:
  apiserverHost: m2-513be7d11ee64ed7826f5c7c4cfdf10c.kubernikus.eu-nl-1.cloud.sap
  wormholeHost: m2-513be7d11ee64ed7826f5c7c4cfdf10c-wormhole.kubernikus.eu-nl-1.cloud.sap

openstack:
  region: eu-nl-1
  lbSubnetID: "4e16419d-eb9f-4245-bcd3-179067c59298"
  routerID: "757e348c-6e10-457c-b418-81933725c077"
```
The api part represents the host of the ingress definition and has to have some form:  
```
<clustername>-<projectid>.kubernikus.<region>.cloud.sap
<clustername>-<projectid>-wormhole.kubernikus.<region>.cloud.sap
```

These openstack things need also be added:  
```region```  the region in which this is running  
```lbSubnetID```  the private _subnet_ to which the load balancer should be added  
```routerID```  the id of your router.  

### The user

The technical user which you use to interact has to have a default project set. This is very problematic as we have no way of changing a normal user which is in LDAP.

Thus the user has to be a technical user and created via commandline.
To be able to do this it has to be created outside of any LDAP configured domain.

```
openstack user create --with --many --options
```
Will help to create a user inside the ```kubernikus``` domain with a default project set to the project id.

```openstack role list``` and ```openstack role add``` will need to be used to get the user the necessary rights inside your target project.
```
admin
kubernetes_admin
member
network_admin
compute_admin
```
Now you should be able to see your tech user in elektra including it's roles.

## Installing

```
helm install --name m2-513be7d11ee64ed7826f5c7c4cfdf10c charts/kube-master/ --values values.yaml  --namespace kubernikus
```
Should now suffice the name is your clustername and the projectid.
