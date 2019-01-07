# Kubernikus

[![Kubernikus](/assets/kubernikus.svg)](https://github.com/sapcc/kubernikus)

[![Build Status](https://travis-ci.org/sapcc/kubernikus.svg?branch=master)](https://travis-ci.org/sapcc/kubernikus)
[![Go Report Card](https://goreportcard.com/badge/github.com/sapcc/kubernikus)](https://goreportcard.com/report/github.com/sapcc/kubernikus)
[![Contributions](https://img.shields.io/badge/contributions-welcome-brightgreen.svg)](https://travis-ci.org/sapcc/kubernikus.svg?branch=master)
[![License](https://img.shields.io/badge/license-Apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.txt)

----

Kubernikus is "Kubernetes as a Service" for Openstack.

It allows to easily manage Kubernetes clusters that are natively integrated with Openstack. The architecture is designed to facilitate the operation as a managed service.

----

## Features

  * Architecured to be operated as a managed service
  * Masters are managed centrally 
  * Nodes are decentralized in customer's projects
  * 100% Vanilla Kubernetes
  * 100% Compatible Openstack API
  * Air-Gapped Masters and Nodes
  * Full TLS encryption between all components
  * Auto-Updating nodes based on CoreOS Container Linux
  * Authentication Tooling 
  * Unified Authorization Policy between Openstack and Kubernetes RBAC
  
## Guiding Principles

  * Running Kubernetes using Kubernetes
  * Automation is driven by Operators
  * Cloud Native Tooling: Golang, Helm, Swagger, Prometheus
  
## Prerequisites

  * Openstack (including LBaaS)
  * Kubernetes Seed-Cluster (1.7+)
  
## Documentation

More documentation can be found at:

  * [Kubernikus Docs](./docs/)

## License
This project is licensed under the Apache2 License - see the [LICENSE](LICENSE) file for details

