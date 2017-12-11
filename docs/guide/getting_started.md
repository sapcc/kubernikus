---
title: Getting Started
weight: -10
---

## Getting Started

[Kubernetes](https://kubernetes.io/) is an open-source system for automating
deployment, scaling, and management of containerized applications.

The "Kubernetes as a Service" offering on Converged Cloud (Codename: Kubernikus)
makes it easy to run Kubernetes clusters that are natively integrated with
OpenStack. It is a managed service that takes care of installing, upgrading and
operating the cluster.

It provides an easy entry to deploy containerized payloads and getting started
with Kubernetes without the operational overhead of setting up  Kubernetes.
Due to the tight and convenient integration with OpenStack it becomes easy to
combine VM and cloud-native workloads.

Running on Converged Cloud opens the possibility to connect with in-house
business/technical/development systems that can’t or shouldn't be exposed to
public clouds.

### Key Features

  * Masters are managed centrally
  * Workload nodes are located in customer's projects
  * Combine VM and containerised payloads

### Enhanced Security

  * Air-Gapped masters and nodes
  * Full TLS encryption between all components
  * Unified authorization policy between OpenStack and [Kubernetes RBAC](http://blog.kubernetes.io/2017/04/rbac-support-in-kubernetes.html)
  * Authentication tooling
  * Auto-Updating nodes based on [CoreOS Container Linux](https://coreos.com/why/)

### Compliance

  * 100% Vanilla Kubernetes
  * 100% Compatible OpenStack API

## Demo

<iframe width="708" height="398" src="https://www.youtube.com/embed/1dPxPU9fHTg" frameborder="0" allowfullscreen></iframe>

### Tech Preview

<span class="label label-info">Note</span> This service is now (Dec 2017) available
in Tech Preview with a limited amount of beta-testers. If you are interested in
trying it out please contact [Michael
Schmidt](mailto:michael02.schmidt@sap.com) with a short description of your use
case.

#### Availability

  * Regions: `eu-nl-1` `na-us-1`
  * Domains: `monsoon3`

#### Restrictions

  * Access needs to be requested on a per-project basis
  * Each project can only contain a single cluster
  * Kubernetes 1.7.5

#### Agreement

The Tech Preview provides early access to a service that is still under
development, enabling you to test functionality and provide feedback. However,
the service may not be functionally complete, and is not yet intended for
production use.

We **cannot guarantee** that clusters created during the Tech Preview can be
carried over into productive use once the service becomes generally available.

~> The standard SLA provided for productive Converged Cloud services does not apply.

### Support

To allow for direct, convenient feedback and support please join the
[#kubernikus-users](https://convergedcloud.slack.com/messages/kubernikus-users)
channel in the [Converged Cloud Slack](https://convergedcloud.slack.com)
workspace. Any SAP employee is allowed to sign up and access this workspace
using the SAP email address.

There's also an open weekly meeting for all users and everyone interested in
Kubernikus. Next dates are being announced and pinned in
[#kubernikus-users](https://convergedcloud.slack.com/messages/kubernikus-users).

There you will also find the
[#kubernetes](https://convergedcloud.slack.com/messages/kubernikus-users)
channel for general topics related to Kubernetes in SAP and specifially on
Converged Cloud.
