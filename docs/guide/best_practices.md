---
title: Best Practices 
---

## Surviving Node-Updates

  * Container-Linux-Update-Orchestrator as seen in Bare-Metal
  
## Automating Authentication Refresh

## Integration for CI Systems

  * Add credentials to a build agent to communicate with a Kubernikus Kluster.

### Set up kubectl an kubernikusctl
First you have to set up `kubectl` and `kubernikusctl` on your build agent. 
Follow the [instructions](https://github.com/sapcc/kubernikus/blob/master/docs/guide/authentication.md#authenticating-with-kubernetes). 

### Create a technical user
To renew certs with `kubernikusctl auth` you have to store your `username` and `password` on the build agent. For privacy reasons you can create and add a technical user instead of your personal one. To create a technical user follow this [guide](https://documentation.global.cloud.sap/docs/support/specific-requests/technical-user-requests.html) (SAP only).

### Add credentials to your environment
Add following variables to your environment on your build agent and fill it with your credentials:

```
OS_AUTH_URL=https://identity-3.eu-nl-1.cloud.sap/v3
OS_USERNAME=T27F923CD2DC8D443 
OS_PASSWORD=abcabc
OS_PROJECT_NAME=testproject
OS_PROJECT_DOMAIN_NAME=monsoon3
```

### Use kubernikusctl into your build job 
Add `kubernikusctl auth init` command to your build job. It will check your environment for the credentials and fetches certificates.

