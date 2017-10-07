# Kubectl Container

This container contains a `kubectl` binary. It will also automatically fetch
short lived certificates from Vault.


## Usage

You need to pass in the `REGION` and an Github access token `GITHUB_TOKEN` in
order to authenticate against Vault.

```
docker run -ti \
  -e REGION=staging \
  -e GITHUB_TOKEN=be76f8004ffb265993c80d81612cea6aa12345 \
  hub.global.cloud.sap/monsoon/kubectl:v.1.4.6 kubectl get pods
```

## Concourse Uage

```
- task: kubectl
  config:
    platform: 'linux'
    image_resource:
      type: docker-image
      source:
        repository: hub.global.cloud.sap/monsoon/kubectl
        tag: v1.4.6
    run:
      path: kubectl
      args:
        - get
        - pods
        - --namespace=monsoon3
    params:
      REGION: staging
      GITHUB_TOKEN: {{GITHUB_TOKEN}}
```
