# Change Log

## [Unreleased](https://github.com/sapcc/kubernikus/tree/HEAD)

[Full Changelog](https://github.com/sapcc/kubernikus/compare/v1.4.0...HEAD)

**Implemented enhancements:**

- apiserver: Improve logs [\#2](https://github.com/sapcc/kubernikus/issues/2)

**Closed issues:**

- Discover all missing attributes via operator [\#85](https://github.com/sapcc/kubernikus/issues/85)

**Merged pull requests:**

- Relax stalebot [\#143](https://github.com/sapcc/kubernikus/pull/143) ([databus23](https://github.com/databus23))
- Correct Test Flag Name [\#142](https://github.com/sapcc/kubernikus/pull/142) ([notque](https://github.com/notque))
- Typo fixes. [\#141](https://github.com/sapcc/kubernikus/pull/141) ([notque](https://github.com/notque))
- Fix Broken Doc Link [\#139](https://github.com/sapcc/kubernikus/pull/139) ([notque](https://github.com/notque))

## [v1.4.0](https://github.com/sapcc/kubernikus/tree/v1.4.0) (2017-12-22)
[Full Changelog](https://github.com/sapcc/kubernikus/compare/v1.3.0...v1.4.0)

**Implemented enhancements:**

- Upgrade to 1.8 [\#62](https://github.com/sapcc/kubernikus/issues/62)
- Kubernetes v1.9.0 Support [\#135](https://github.com/sapcc/kubernikus/pull/135) ([BugRoger](https://github.com/BugRoger))

**Fixed bugs:**

- Get cluster that doesn't exist gives 502 instead of 404 [\#133](https://github.com/sapcc/kubernikus/issues/133)

**Merged pull requests:**

- fix default backend and use it only for 502,503,504 [\#134](https://github.com/sapcc/kubernikus/pull/134) ([auhlig](https://github.com/auhlig))

## [v1.3.0](https://github.com/sapcc/kubernikus/tree/v1.3.0) (2017-12-20)
[Full Changelog](https://github.com/sapcc/kubernikus/compare/v1.2.0...v1.3.0)

**Implemented enhancements:**

- Bubble Up Events Log [\#47](https://github.com/sapcc/kubernikus/issues/47)

**Closed issues:**

- Refactor Logging [\#100](https://github.com/sapcc/kubernikus/issues/100)
- End-User Docs for Kubernetes Dashboard [\#98](https://github.com/sapcc/kubernikus/issues/98)

**Merged pull requests:**

- Upgrade to Kubernetes 1.8.5 [\#132](https://github.com/sapcc/kubernikus/pull/132) ([BugRoger](https://github.com/BugRoger))
- Add leveled logging [\#129](https://github.com/sapcc/kubernikus/pull/129) ([databus23](https://github.com/databus23))
- Remove migration to retrofit security group id to existing clusters [\#128](https://github.com/sapcc/kubernikus/pull/128) ([databus23](https://github.com/databus23))
- Specify security groups by id when creating servers [\#126](https://github.com/sapcc/kubernikus/pull/126) ([databus23](https://github.com/databus23))
- removes glog  [\#125](https://github.com/sapcc/kubernikus/pull/125) ([BugRoger](https://github.com/BugRoger))
- Update to go-swagger 0.13 [\#120](https://github.com/sapcc/kubernikus/pull/120) ([databus23](https://github.com/databus23))
- Request-ID Tracing [\#118](https://github.com/sapcc/kubernikus/pull/118) ([BugRoger](https://github.com/BugRoger))
- enforce holy import trinity using goimports [\#117](https://github.com/sapcc/kubernikus/pull/117) ([databus23](https://github.com/databus23))

## [v1.2.0](https://github.com/sapcc/kubernikus/tree/v1.2.0) (2017-12-11)
[Full Changelog](https://github.com/sapcc/kubernikus/compare/v1.1.0...v1.2.0)

**Implemented enhancements:**

- Openstack Metadata Digester [\#101](https://github.com/sapcc/kubernikus/issues/101)
- Stricter Pool/Node Detection [\#79](https://github.com/sapcc/kubernikus/issues/79)
- Darwin Build for Kubernikusctl [\#77](https://github.com/sapcc/kubernikus/issues/77)
- Deletion of Errored Klusters Fails [\#75](https://github.com/sapcc/kubernikus/issues/75)
- Authentication Subsystem [\#60](https://github.com/sapcc/kubernikus/issues/60)

**Closed issues:**

- Test Transpiling of Ignition Templates [\#94](https://github.com/sapcc/kubernikus/issues/94)
- Travis / Slack Integration Broken [\#88](https://github.com/sapcc/kubernikus/issues/88)
- Ingress Default Backend doesn't know JSON  [\#80](https://github.com/sapcc/kubernikus/issues/80)
- Validate if imcp redirects are working [\#65](https://github.com/sapcc/kubernikus/issues/65)
- Generate swagger.yaml from Spec [\#53](https://github.com/sapcc/kubernikus/issues/53)
- Add README [\#46](https://github.com/sapcc/kubernikus/issues/46)

**Merged pull requests:**

- Allow pods to ping outside using masquerade [\#115](https://github.com/sapcc/kubernikus/pull/115) ([SchwarzM](https://github.com/SchwarzM))
- implements structured logging for apiserver [\#113](https://github.com/sapcc/kubernikus/pull/113) ([BugRoger](https://github.com/BugRoger))
- Masquerading in static firewall deployment [\#111](https://github.com/sapcc/kubernikus/pull/111) ([SchwarzM](https://github.com/SchwarzM))
- phase transition summary [\#108](https://github.com/sapcc/kubernikus/pull/108) ([auhlig](https://github.com/auhlig))
- Logging Prototype [\#106](https://github.com/sapcc/kubernikus/pull/106) ([BugRoger](https://github.com/BugRoger))
- rename and add some more metrics. also add metric unit test [\#105](https://github.com/sapcc/kubernikus/pull/105) ([auhlig](https://github.com/auhlig))
- Nginx ingress [\#104](https://github.com/sapcc/kubernikus/pull/104) ([auhlig](https://github.com/auhlig))
- Node exporter [\#103](https://github.com/sapcc/kubernikus/pull/103) ([auhlig](https://github.com/auhlig))
- First throw at a bit doku of how to dev the helm charts [\#102](https://github.com/sapcc/kubernikus/pull/102) ([SchwarzM](https://github.com/SchwarzM))
- introducing basic metrics [\#96](https://github.com/sapcc/kubernikus/pull/96) ([auhlig](https://github.com/auhlig))
- initial prometheus2 [\#95](https://github.com/sapcc/kubernikus/pull/95) ([auhlig](https://github.com/auhlig))
- unify kluster specs [\#93](https://github.com/sapcc/kubernikus/pull/93) ([databus23](https://github.com/databus23))
- Tests e2e [\#89](https://github.com/sapcc/kubernikus/pull/89) ([auhlig](https://github.com/auhlig))

## [v1.1.0](https://github.com/sapcc/kubernikus/tree/v1.1.0) (2017-11-06)
[Full Changelog](https://github.com/sapcc/kubernikus/compare/v1.0.0...v1.1.0)

**Implemented enhancements:**

- Setup CoreDNS for kubernikus-system [\#70](https://github.com/sapcc/kubernikus/issues/70)
- Log/Tracing Utility [\#24](https://github.com/sapcc/kubernikus/issues/24)
- Configurable Defaults [\#21](https://github.com/sapcc/kubernikus/issues/21)

**Fixed bugs:**

- Switch Etcd Volume AccessMode [\#71](https://github.com/sapcc/kubernikus/issues/71)
- Bootstrapped Node Certificate Gets Deleted on Reboot [\#41](https://github.com/sapcc/kubernikus/issues/41)
- Openstack Client Cache Doesn't Invalide Deleted Klusters [\#37](https://github.com/sapcc/kubernikus/issues/37)

**Closed issues:**

- Add confirmation button the cluster delete in elektra [\#86](https://github.com/sapcc/kubernikus/issues/86)
- Seed Cinder Default Storage Class [\#69](https://github.com/sapcc/kubernikus/issues/69)
- Move secrets from Cluster TPR to secret [\#67](https://github.com/sapcc/kubernikus/issues/67)
- Kubernikus Logo [\#61](https://github.com/sapcc/kubernikus/issues/61)
- Extend Continuous Delivery Pipeline [\#59](https://github.com/sapcc/kubernikus/issues/59)
- apiserver reachability from pods [\#57](https://github.com/sapcc/kubernikus/issues/57)
- Bad Gateway on Deployment [\#55](https://github.com/sapcc/kubernikus/issues/55)
- Sane Infrastructure Setup [\#54](https://github.com/sapcc/kubernikus/issues/54)
- Github Workflow [\#50](https://github.com/sapcc/kubernikus/issues/50)
- RKT Pods for Kubelet + Wormhole Client [\#44](https://github.com/sapcc/kubernikus/issues/44)
- Cleanup and enhance spec [\#3](https://github.com/sapcc/kubernikus/issues/3)

**Merged pull requests:**

- Update to go-swagger 0.12.0 [\#91](https://github.com/sapcc/kubernikus/pull/91) ([databus23](https://github.com/databus23))
- Expose service and cluster CIDR in the api [\#90](https://github.com/sapcc/kubernikus/pull/90) ([databus23](https://github.com/databus23))
- Switch from TPR to CRD [\#87](https://github.com/sapcc/kubernikus/pull/87) ([databus23](https://github.com/databus23))
- add note on how to install kubernikusctl [\#84](https://github.com/sapcc/kubernikus/pull/84) ([auhlig](https://github.com/auhlig))
- update vp chart [\#83](https://github.com/sapcc/kubernikus/pull/83) ([auhlig](https://github.com/auhlig))
- Prevent deletion of kluster as long as nodepools are present [\#81](https://github.com/sapcc/kubernikus/pull/81) ([SchwarzM](https://github.com/SchwarzM))
- Kctlread [\#76](https://github.com/sapcc/kubernikus/pull/76) ([SchwarzM](https://github.com/SchwarzM))
- caches cors preflight request for 10 min [\#68](https://github.com/sapcc/kubernikus/pull/68) ([edda](https://github.com/edda))

## [v1.0.0](https://github.com/sapcc/kubernikus/tree/v1.0.0) (2017-10-04)
**Implemented enhancements:**

- Docker Options Dropin [\#64](https://github.com/sapcc/kubernikus/issues/64)
- Add Kube-Proxy to Nodes [\#38](https://github.com/sapcc/kubernikus/issues/38)
- Seed ClusterRoleBindings [\#35](https://github.com/sapcc/kubernikus/issues/35)
- Add Kube-Proxy to Nodes [\#34](https://github.com/sapcc/kubernikus/issues/34)
- Expose NodePool CRUD via API [\#31](https://github.com/sapcc/kubernikus/issues/31)

**Fixed bugs:**

- Fix Deseeding of Service User [\#58](https://github.com/sapcc/kubernikus/issues/58)
- Deleting a Kluster via API Fails [\#36](https://github.com/sapcc/kubernikus/issues/36)
- Spawns Too Many Nodes [\#28](https://github.com/sapcc/kubernikus/issues/28)

**Closed issues:**

- Improve NodeAPI [\#49](https://github.com/sapcc/kubernikus/issues/49)
- Remove Dependency OpenstackSeeder [\#48](https://github.com/sapcc/kubernikus/issues/48)
- Kube-Proxy br\_netfilter Missing [\#42](https://github.com/sapcc/kubernikus/issues/42)
- Cluster-State Aware LaunchController  [\#25](https://github.com/sapcc/kubernikus/issues/25)
- Kluster persistence [\#18](https://github.com/sapcc/kubernikus/issues/18)
- Implement cluster edit [\#17](https://github.com/sapcc/kubernikus/issues/17)
- Add CORS support [\#16](https://github.com/sapcc/kubernikus/issues/16)
- What about testing? [\#11](https://github.com/sapcc/kubernikus/issues/11)
- Detect when kluster is ready [\#9](https://github.com/sapcc/kubernikus/issues/9)
- Add API call for getting kluster credentials [\#8](https://github.com/sapcc/kubernikus/issues/8)
- Implement nodes controller [\#6](https://github.com/sapcc/kubernikus/issues/6)
- Implement kluster deletion [\#5](https://github.com/sapcc/kubernikus/issues/5)
- Openstack CloudProvider Reauth [\#1](https://github.com/sapcc/kubernikus/issues/1)

**Merged pull requests:**

- add charts for k8sniff [\#39](https://github.com/sapcc/kubernikus/pull/39) ([auhlig](https://github.com/auhlig))
- simplify kube-master chart [\#19](https://github.com/sapcc/kubernikus/pull/19) ([databus23](https://github.com/databus23))
- Cluster edit\(patch\) and delete [\#15](https://github.com/sapcc/kubernikus/pull/15) ([auhlig](https://github.com/auhlig))
- CI [\#14](https://github.com/sapcc/kubernikus/pull/14) ([auhlig](https://github.com/auhlig))



\* *This Change Log was automatically generated by [github_changelog_generator](https://github.com/skywinder/Github-Changelog-Generator)*