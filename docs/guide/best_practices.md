---
title: Best Practices 
---

## Surviving Node-Updates

  * Container-Linux-Update-Orchestrator as seen in Bare-Metal
  
## Automating Authentication Refresh

## Integration for CI Systems

  * Add credentials to a build agent to communicate with a Kubernikus Kluster.

### Set up kubectl an kubernikusctl
First you have to install `kubectl` and `kubernikusctl` on your build agent. 
Here you can find the [instructions](https://github.com/sapcc/kubernikus/blob/master/docs/guide/authentication.md#authenticating-with-kubernetes). 

### Add credentials to your environment
`kubernikusctl` will update your certificates since they expire daily. Within your build you can add environment variables which kubernikusctl will use to authenticate against the Kubernikus Kluster. Add `Auth Url`, `Username`, `Password`, `User Domain`, `Project name` and `Project Domain` to your environment inside the build. 

It should look like this:

```
OS_AUTH_URL=https://identity...../v3 
OS_USERNAME=T27F923CD2DC8D443 
OS_PASSWORD=abcabc
OS_USER_DOMAIN_NAME=monsoon3
OS_PROJECT_NAME=testproject
OS_PROJECT_DOMAIN_NAME=monsoon3
```

### Use kubernikusctl inside your build job 
After setting up your environment you can add the `kubernikusctl auth init` command to your build job. It will look for credentials and fetches certificates. 

### Create a technical user (SAP)
If you would like to avoid using your own `username` and `password` on a build agent you can create a technical user instead. Follow the instructions at the SAP Converged Cloud Documentation.