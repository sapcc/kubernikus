Kubernikus E2E Tests
--------------------

The E2E integration test suite covers API and smoke tests to verify basic functionality of Kubernikus. 
It consists of 2 parts:
  * API tests:  covers CRUD actions using API
  * Smoke test: covers network test and attachment of persistent volumes

Per default all tests are executed. However tests are grouped and can be run independently. See section [Usage](Usage) .

### Usage

Before using the integration test suite the following the of parameters must be provided via configuration file or local environment.

(1) Via configuration file found here `test/e2e/e2e_config.yaml`:
```
kubernikus_api_server:  "kubernikus.<region_name>.cloud.sap"
kubernikus_api_version: "v1"

openstack:
  project_name:         <project_name>
  project_domain_name:  <project_domain_name>
  username:             <username>
  password:             <password>
  user_domain_name:     <user_domain_name>
  region_name:          <region_name>
  auth_url:             <auth_url>
```

(2) Via local environment:
```
export KUBERNIKUS_API_SERVER="kubernikus.<region_name>.cloud.sap"
export KUBERNIKUS_API_VERSION="v1
export OS_PROJECT_NAME=<project_name>
export OS_PROJECT_DOMAIN_NAME=<project_domain_name>
export OS_USERNAME=<username>
export OS_PASSWORD=<password>
export OS_USER_DOMAIN_NAME=<user_domain_name>
export OS_REGION_NAME=<region_name>
export OS_AUTH_URL=<auth_url>
```

Invoke the integration tests via the Makefile in the root directory of the Kubernikus project.
```
make test-e2e
``` 
Triggering a single or multiple phases as shown below:
```
./test/e2e/test.sh --<phase>
```
Available phases are:
```
create  - create a new cluster
api     - run API tests
smoke   - run smoke tests
network - run network tests
volume  - run persistent volume tests
delete  - delete the cluster
```
