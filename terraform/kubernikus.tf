provider "openstack" {
  auth_url         = "https://identity-3.${var.region}.cloud.sap/v3"
  region           = "${var.region}"
  user_name        = "${var.user_name}"
  user_domain_name = "${var.user_domain_name}"
  password         = "${var.password}"
  tenant_name      = "cloud_admin"
  domain_name      = "ccadmin"
}

provider "openstack" {
  alias            = "master"

  auth_url         = "https://identity-3.${var.region}.cloud.sap/v3"
  region           = "${var.region}"
  user_name        = "${var.user_name}"
  user_domain_name = "${var.user_domain_name}"
  password         = "${var.password}"
  tenant_name      = "master"
  domain_name      = "ccadmin"
}

provider "openstack" {
  alias            = "master.na-us-1"

  auth_url         = "https://identity-3.na-us-1.cloud.sap/v3"
  region           = "na-us-1"
  user_name        = "${var.user_name}"
  user_domain_name = "${var.user_domain_name}"
  password         = "${var.password}"
  tenant_name      = "master"
  domain_name      = "ccadmin"
}

provider "ccloud" {
  alias            = "cloud_admin"

  auth_url         = "https://identity-3.${var.region}.cloud.sap/v3"
  region           = "${var.region}"
  user_name        = "${var.user_name}"
  user_domain_name = "${var.user_domain_name}"
  password         = "${var.password}"
  tenant_name      = "cloud_admin"
  domain_name      = "ccadmin"
}

terraform {
  backend "swift" {
    tenant_name       = "master"
    domain_name       = "ccadmin"
    container         = "kubernikus_terraform_state"
    archive_container = "kubernikus_terraform_archive"
    expire_after      = "365d"
  }
}

data "openstack_identity_project_v3" "ccadmin" {
  name      = "ccadmin"
  is_domain = true
}

data "openstack_identity_project_v3" "default" {
  name      = "Default"
  is_domain = true
}

data "openstack_identity_project_v3" "cloud_admin" {
  name      = "cloud_admin"
  domain_id = "${data.openstack_identity_project_v3.ccadmin.id}"
}

data "openstack_identity_group_v3" "ccadmin_domain_admins" {
  name = "CCADMIN_DOMAIN_ADMINS"
}

data "openstack_identity_user_v3" "kubernikus_terraform" {
  name      = "kubernikus-terraform"
  domain_id = "${data.openstack_identity_project_v3.default.id}"
}

data "openstack_identity_role_v3" "admin" {
  name = "admin"
}

data "openstack_identity_role_v3" "member" {
  name = "member"
}

data "openstack_identity_role_v3" "compute_admin" {
  name = "compute_admin"
}

data "openstack_identity_role_v3" "network_admin" {
  name = "network_admin"
}

data "openstack_identity_role_v3" "resource_admin" {
  name = "resource_admin"
}

data "openstack_identity_role_v3" "volume_admin" {
  name = "volume_admin"
}

data "openstack_identity_role_v3" "swiftoperator" {
  name = "swiftoperator"
}
data "openstack_identity_role_v3" "swiftreseller" {
  name = "swiftreseller"
}

data "openstack_identity_role_v3" "cloud_compute_admin" {
  name = "cloud_compute_admin"
}

data "openstack_identity_role_v3" "cloud_dns_admin" {
  name = "cloud_dns_admin"
}

data "openstack_identity_role_v3" "cloud_image_admin" {
  name = "cloud_image_admin"
}

data "openstack_identity_role_v3" "cloud_keymanager_admin" {
  name = "cloud_keymanager_admin"
}

data "openstack_identity_role_v3" "cloud_network_admin" {
  name = "cloud_network_admin"
}

data "openstack_identity_role_v3" "cloud_resource_admin" {
  name = "cloud_resource_admin"
}

data "openstack_identity_role_v3" "cloud_sharedfilesystem_admin" {
  name = "cloud_sharedfilesystem_admin"
}

data "openstack_identity_role_v3" "cloud_volume_admin" {
  name = "cloud_volume_admin"
}

data "openstack_networking_network_v2" "external" {
  name = "FloatingIP-external-ccadmin"
}

data "openstack_networking_network_v2" "external_e2e" {
  name = "FloatingIP-external-monsoon3-01"
}

resource "openstack_identity_role_v3" "kubernetes_admin" {
  name = "kubernetes_admin"
}

resource "openstack_identity_role_v3" "kubernetes_member" {
  name = "kubernetes_member"
}

resource "openstack_identity_user_v3" "kubernikus_pipeline" {
  domain_id    = "${data.openstack_identity_project_v3.default.id}"
  name         = "kubernikus-pipeline"
  description  = "Kubernikus Pipeline User"
  password     = "${var.kubernikus-pipeline-password}"

  ignore_change_password_upon_first_use = true
  ignore_password_expiry = true
}

resource "openstack_identity_user_v3" "kubernikus_service" {
  domain_id    = "${data.openstack_identity_project_v3.default.id}"
  name         = "kubernikus"
  description  = "Kubernikus Service User"
  password     = "${var.kubernikus-service-password}"

  ignore_change_password_upon_first_use = true
  ignore_password_expiry = true
}



resource "openstack_identity_project_v3" "kubernikus" {
  name        = "kubernikus"
  domain_id   = "${data.openstack_identity_project_v3.ccadmin.id}"
  description = "Kubernikus Control-Plane"
}

resource "openstack_identity_role_assignment_v3" "admin" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus.id}"
  role_id    = "${data.openstack_identity_role_v3.admin.id}"
}

resource "openstack_identity_role_assignment_v3" "compute_admin" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus.id}"
  role_id    = "${data.openstack_identity_role_v3.compute_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "network_admin" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus.id}"
  role_id    = "${data.openstack_identity_role_v3.network_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "resource_admin" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus.id}"
  role_id    = "${data.openstack_identity_role_v3.resource_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "volume_admin" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus.id}"
  role_id    = "${data.openstack_identity_role_v3.volume_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernetes_admin" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus.id}"
  role_id    = "${openstack_identity_role_v3.kubernetes_admin.id}"
}


resource "openstack_identity_role_assignment_v3" "terraform_kubernetes_admin" {
  user_id    = "${data.openstack_identity_user_v3.kubernikus_terraform.id}"
  project_id = "${openstack_identity_project_v3.kubernikus.id}"
  role_id    = "${openstack_identity_role_v3.kubernetes_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "terraform_member" {
  user_id    = "${data.openstack_identity_user_v3.kubernikus_terraform.id}"
  project_id = "${openstack_identity_project_v3.kubernikus.id}"
  role_id    = "${data.openstack_identity_role_v3.member.id}"
}

resource "openstack_identity_role_assignment_v3" "pipeline_kubernetes_admin" {
  user_id    = "${openstack_identity_user_v3.kubernikus_pipeline.id}"
  project_id = "${openstack_identity_project_v3.kubernikus.id}"
  role_id    = "${openstack_identity_role_v3.kubernetes_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernikus-admin" {
  user_id    = "${openstack_identity_user_v3.kubernikus_service.id}"
  project_id = "${data.openstack_identity_project_v3.cloud_admin.id}"
  role_id    = "${data.openstack_identity_role_v3.admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernikus-cloud_compute_admin" {
  user_id    = "${openstack_identity_user_v3.kubernikus_service.id}"
  project_id = "${data.openstack_identity_project_v3.cloud_admin.id}"
  role_id    = "${data.openstack_identity_role_v3.cloud_compute_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernikus-cloud_dns_admin" {
  user_id    = "${openstack_identity_user_v3.kubernikus_service.id}"
  project_id = "${data.openstack_identity_project_v3.cloud_admin.id}"
  role_id    = "${data.openstack_identity_role_v3.cloud_dns_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernikus-cloud_image_admin" {
  user_id    = "${openstack_identity_user_v3.kubernikus_service.id}"
  project_id = "${data.openstack_identity_project_v3.cloud_admin.id}"
  role_id    = "${data.openstack_identity_role_v3.cloud_image_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernikus-cloud_keymanager_admin" {
  user_id    = "${openstack_identity_user_v3.kubernikus_service.id}"
  project_id = "${data.openstack_identity_project_v3.cloud_admin.id}"
  role_id    = "${data.openstack_identity_role_v3.cloud_keymanager_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernikus-cloud_network_admin" {
  user_id    = "${openstack_identity_user_v3.kubernikus_service.id}"
  project_id = "${data.openstack_identity_project_v3.cloud_admin.id}"
  role_id    = "${data.openstack_identity_role_v3.cloud_network_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernikus-cloud_resource_admin" {
  user_id    = "${openstack_identity_user_v3.kubernikus_service.id}"
  project_id = "${data.openstack_identity_project_v3.cloud_admin.id}"
  role_id    = "${data.openstack_identity_role_v3.cloud_resource_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernikus-cloud_sharedfilesystem_admin" {
  user_id    = "${openstack_identity_user_v3.kubernikus_service.id}"
  project_id = "${data.openstack_identity_project_v3.cloud_admin.id}"
  role_id    = "${data.openstack_identity_role_v3.cloud_sharedfilesystem_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernikus-cloud_volume_admin" {
  user_id    = "${openstack_identity_user_v3.kubernikus_service.id}"
  project_id = "${data.openstack_identity_project_v3.cloud_admin.id}"
  role_id    = "${data.openstack_identity_role_v3.cloud_volume_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernikus-swiftreseller" {
  user_id    = "${openstack_identity_user_v3.kubernikus_service.id}"
  project_id = "${data.openstack_identity_project_v3.cloud_admin.id}"
  role_id    = "${data.openstack_identity_role_v3.swiftreseller.id}"
}





resource "ccloud_quota" "kubernikus" {
  provider = "ccloud.cloud_admin" 

  domain_id  = "${data.openstack_identity_project_v3.ccadmin.id}"
  project_id = "${openstack_identity_project_v3.kubernikus.id}"

  compute {
    instances = 10
    cores     = 48 
    ram       = 81920 
  }

  volumev2 {
    capacity  = 1024
    snapshots = 0
    volumes   = 100
  }

  network {
		floating_ips         = 4
		networks             = 1
		ports                = 500
		routers              = 2
		security_group_rules = 64
		security_groups      = 4
		subnets              = 1
		healthmonitors       = 10
		l7policies           = 10
		listeners            = 10
		loadbalancers        = 10
		pools                = 10
    pool_members         = 10
  }

  objectstore {
    capacity = 1073741824
  }
}

resource "openstack_networking_rbacpolicies_v2" "external" {
  action        = "access_as_shared"
  object_id     = "${data.openstack_networking_network_v2.external.id}"
  object_type   = "network"
  target_tenant = "${openstack_identity_project_v3.kubernikus.id}"
}

resource "openstack_networking_network_v2" "network" {
  tenant_id      = "${openstack_identity_project_v3.kubernikus.id}"
  name           = "kubernikus"
  admin_state_up = "true"
  depends_on     = ["ccloud_quota.kubernikus"]
}

resource "openstack_networking_subnet_v2" "subnet" {
  tenant_id  = "${openstack_identity_project_v3.kubernikus.id}"
  name       = "kubernikus"
  network_id = "${openstack_networking_network_v2.network.id}"
  cidr       = "198.18.0.0/24"
  ip_version = 4
}

resource "openstack_networking_router_v2" "router" {
  tenant_id           = "${openstack_identity_project_v3.kubernikus.id}"
  name                = "kubernikus"
  admin_state_up      = true
  external_network_id = "${data.openstack_networking_network_v2.external.id}"
  depends_on          = ["ccloud_quota.kubernikus"]
}

resource "openstack_networking_router_interface_v2" "router_interface" {
  router_id = "${openstack_networking_router_v2.router.id}"
  subnet_id = "${openstack_networking_subnet_v2.subnet.id}"
}


data "openstack_networking_secgroup_v2" "kubernikus_default" {
  name        = "default"
  tenant_id   = "${openstack_identity_project_v3.kubernikus.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_0" {
  tenant_id = "${openstack_identity_project_v3.kubernikus.id}"
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "tcp"
  remote_ip_prefix  = "198.18.0.0/15"
  security_group_id = "${data.openstack_networking_secgroup_v2.kubernikus_default.id}"

  depends_on          = ["ccloud_quota.kubernikus"]
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_1" {
  tenant_id = "${openstack_identity_project_v3.kubernikus.id}"
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "udp"
  remote_ip_prefix  = "198.18.0.0/15"
  security_group_id = "${data.openstack_networking_secgroup_v2.kubernikus_default.id}"

  depends_on          = ["ccloud_quota.kubernikus"]
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_ssh" {
  tenant_id = "${openstack_identity_project_v3.kubernikus.id}"
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "tcp"
  remote_ip_prefix  = "198.18.0.0/24"
  port_range_min    = 22
  port_range_max    = 22
  security_group_id = "${data.openstack_networking_secgroup_v2.kubernikus_default.id}"

  depends_on          = ["ccloud_quota.kubernikus"]
}

resource "openstack_identity_service_v3" "kubernikus" {
  name        = "kubernikus"
  type        = "kubernikus"
  description = "End-User Kubernikus Service"
}

resource "openstack_identity_service_v3" "kubernikus-kubernikus" {
  name        = "kubernikus"
  type        = "kubernikus-kubernikus"
  description = "Admin Kubernikus Service"
}

resource "openstack_identity_endpoint_v3" "kubernikus" {
  service_id = "${openstack_identity_service_v3.kubernikus.id}"
  name       = "kubernikus"
  interface  = "public"
  region     = "${var.region}"
  url        = "https://kubernikus.${var.region}.cloud.sap"
}

resource "openstack_identity_endpoint_v3" "kubernikus-kubernikus" {
  service_id = "${openstack_identity_service_v3.kubernikus-kubernikus.id}"
  name       = "kubernikus-kubernikus"
  interface  = "public"
  region     = "${var.region}"
  url        = "https://k-${var.region}.admin.cloud.sap"
}




data "openstack_dns_zone_v2" "region_cloud_sap" {
  provider = "openstack.master"
  name     = "${var.region}.cloud.sap."
}

data "openstack_dns_zone_v2" "admin_cloud_sap" {
  provider = "openstack.master.na-us-1"
  name     = "admin.cloud.sap."
}

resource "openstack_dns_recordset_v2" "kubernikus-ingress" {
  provider = "openstack.master"
  zone_id = "${data.openstack_dns_zone_v2.region_cloud_sap.id}"
  name    = "kubernikus-ingress.${var.region}.cloud.sap."
  type    = "A"
  ttl     = 1800
  records = ["${var.lb-kubernikus-ingress-fip}"]
}

resource "openstack_dns_recordset_v2" "kubernikus-k8sniff" {
  provider = "openstack.master"
  zone_id = "${data.openstack_dns_zone_v2.region_cloud_sap.id}"
  name    = "kubernikus-k8sniff.${var.region}.cloud.sap."
  type    = "A"
  ttl     = 1800
  records = ["${var.lb-kubernikus-k8sniff-fip}"]
}

resource "openstack_dns_recordset_v2" "wildcard-kubernikus" {
  provider = "openstack.master"
  zone_id = "${data.openstack_dns_zone_v2.region_cloud_sap.id}"
  name    = "*.kubernikus.${var.region}.cloud.sap."
  type    = "CNAME"
  ttl     = 1800
  records = ["kubernikus-k8sniff.${var.region}.cloud.sap."]
}

resource "openstack_dns_recordset_v2" "kubernikus" {
  provider = "openstack.master"
  zone_id = "${data.openstack_dns_zone_v2.region_cloud_sap.id}"
  name    = "kubernikus.${var.region}.cloud.sap."
  type    = "CNAME"
  ttl     = 1800
  records = ["kubernikus-ingress.${var.region}.cloud.sap."]
}

resource "openstack_dns_recordset_v2" "prometheus" {
  provider = "openstack.master"
  zone_id = "${data.openstack_dns_zone_v2.region_cloud_sap.id}"
  name    = "prometheus.kubernikus.${var.region}.cloud.sap."
  type    = "CNAME"
  ttl     = 1800
  records = ["kubernikus-ingress.${var.region}.cloud.sap."]
}

resource "openstack_dns_recordset_v2" "grafana" {
  provider = "openstack.master"
  zone_id = "${data.openstack_dns_zone_v2.region_cloud_sap.id}"
  name    = "grafana.kubernikus.${var.region}.cloud.sap."
  type    = "CNAME"
  ttl     = 1800
  records = ["kubernikus-ingress.${var.region}.cloud.sap."]
}

resource "openstack_dns_recordset_v2" "k-region" {
  provider = "openstack.master.na-us-1"
  zone_id = "${data.openstack_dns_zone_v2.admin_cloud_sap.id}"
  name    = "k-${var.region}.admin.cloud.sap."
  type    = "CNAME"
  ttl     = 1800
  records = ["ingress.admin.cloud.sap."]
}

resource "openstack_dns_recordset_v2" "wildcard-k-region" {
  provider = "openstack.master.na-us-1"
  zone_id = "${data.openstack_dns_zone_v2.admin_cloud_sap.id}"
  name    = "*.k-${var.region}.admin.cloud.sap."
  type    = "CNAME"
  ttl     = 1800
  records = ["kubernikus.admin.cloud.sap."]
}




provider "ccloud" {
  alias            = "kubernikus"

  auth_url         = "https://identity-3.${var.region}.cloud.sap/v3"
  region           = "${var.region}"
  user_name        = "kubernikus-terraform"
  user_domain_name = "Default"
  password         = "${var.password}"
  tenant_id        = "${openstack_identity_project_v3.kubernikus.id}"
}

resource "ccloud_kubernetes" "kluster" {
  provider = "ccloud.kubernikus" 

  is_admin       = true
  name           = "k-${var.region}"
  cluster_cidr   = "198.19.0.0/16"
  service_cidr   = "192.168.128.0/17"
  ssh_public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCXIxVEUgtUVkvk2VM1hmIb8MxvxsmvYoiq9OBy3J8akTGNybqKsA2uhcwxSJX5Cn3si8kfMfka9EWiJT+e1ybvtsGILO5XRZPxyhYzexwb3TcALwc3LuzpF3Z/Dg2jYTRELTGhYmyca3mxzTlCjNXvYayLNedjJ8fIBzoCuSXNqDRToHru7h0Glz+wtuE74mNkOiXSvhtuJtJs7VCNVjobFQNfC1aeDsri2bPRHJJZJ0QF4LLYSayMEz3lVwIDyAviQR2Aa97WfuXiofiAemfGqiH47Kq6b8X7j3bOYGBvJKMUV7XeWhGsskAmTsvvnFxkc5PAD3Ct+liULjiQWlzDrmpTE8aMqLK4l0YQw7/8iRVz6gli42iEc2ZG56ob1ErpTLAKFWyCNOebZuGoygdEQaGTIIunAncXg5Rz07TdPl0Tf5ZZLpiAgR5ck0H1SETnjDTZ/S83CiVZWJgmCpu8YOKWyYRD4orWwdnA77L4+ixeojLIhEoNL8KlBgsP9Twx+fFMWLfxMmiuX+yksM6Hu+Lsm+Ao7Q284VPp36EB1rxP1JM7HCiEOEm50Jb6hNKjgN4aoLhG5yg+GnDhwCZqUwcRJo1bWtm3QvRA+rzrGZkId4EY3cyOK5QnYV5+24x93Ex0UspHMn7HGsHUESsVeV0fLqlfXyd2RbHTmDMP6w=="

  node_pools = [
    { name = "payload", flavor = "m1.xlarge_cpu", size = 3 },
  ]

  depends_on = [
    "openstack_identity_endpoint_v3.kubernikus", 
    "openstack_networking_router_v2.router"
  ]

  lifecycle {
    prevent_destroy = true
  }
}



resource "openstack_identity_project_v3" "kubernikus_e2e" {
  name        = "kubernikus_e2e"
  domain_id   = "${data.openstack_identity_project_v3.ccadmin.id}"
  description = "Kubernikus E2E Tests"
}

resource "openstack_identity_role_assignment_v3" "admin_e2e" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  role_id    = "${data.openstack_identity_role_v3.admin.id}"
}

resource "openstack_identity_role_assignment_v3" "compute_admin_e2e" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  role_id    = "${data.openstack_identity_role_v3.compute_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "network_admin_e2e" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  role_id    = "${data.openstack_identity_role_v3.network_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "resource_admin_e2e" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  role_id    = "${data.openstack_identity_role_v3.resource_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "volume_admin_e2e" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  role_id    = "${data.openstack_identity_role_v3.volume_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "kubernetes_admin_e2e" {
  group_id   = "${data.openstack_identity_group_v3.ccadmin_domain_admins.id}"
  project_id = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  role_id    = "${openstack_identity_role_v3.kubernetes_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "pipeline_kubernetes_admin_e2e" {
  user_id    = "${openstack_identity_user_v3.kubernikus_pipeline.id}"
  project_id = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  role_id    = "${openstack_identity_role_v3.kubernetes_admin.id}"
}

resource "openstack_identity_role_assignment_v3" "pipeline_kubernetes_member_e2e" {
  user_id    = "${openstack_identity_user_v3.kubernikus_pipeline.id}"
  project_id = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  role_id    = "${data.openstack_identity_role_v3.member.id}"
}

resource "openstack_identity_role_assignment_v3" "pipeline_swiftoperator_e2e" {
  user_id    = "${openstack_identity_user_v3.kubernikus_pipeline.id}"
  project_id = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  role_id    = "${data.openstack_identity_role_v3.swiftoperator.id}"
}

resource "ccloud_quota" "kubernikus_e2e" {
  provider = "ccloud.cloud_admin" 

  domain_id  = "${data.openstack_identity_project_v3.ccadmin.id}"
  project_id = "${openstack_identity_project_v3.kubernikus_e2e.id}"

  compute {
    instances = 5 
    cores     = 32 
    ram       = 8192
  }

  volumev2 {
    capacity  = 16 
    snapshots = 2
    volumes   = 2 
  }

  network {
		floating_ips         = 2
		networks             = 1
		ports                = 500
		routers              = 1
		security_group_rules = 64
		security_groups      = 4
		subnets              = 1
		healthmonitors       = 0
		l7policies           = 0
		listeners            = 0
		loadbalancers        = 0
		pools                = 0
  }

  objectstore {
    capacity = 104857600
  }
}


resource "openstack_networking_rbacpolicies_v2" "external_e2e" {
  action        = "access_as_shared"
  object_id     = "${data.openstack_networking_network_v2.external_e2e.id}"
  object_type   = "network"
  target_tenant = "${openstack_identity_project_v3.kubernikus_e2e.id}"
}

resource "openstack_networking_network_v2" "network_e2e" {
  tenant_id      = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  name           = "kubernikus_e2e"
  admin_state_up = "true"
  depends_on     = ["ccloud_quota.kubernikus_e2e"]
}

resource "openstack_networking_subnet_v2" "subnet_e2e" {
  tenant_id  = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  name       = "default"
  network_id = "${openstack_networking_network_v2.network_e2e.id}"
  cidr       = "10.180.0.0/16"
  ip_version = 4
  depends_on = ["ccloud_quota.kubernikus_e2e"]
}

resource "openstack_networking_router_v2" "router_e2e" {
  tenant_id           = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  name                = "default"
  admin_state_up      = true
  external_network_id = "${data.openstack_networking_network_v2.external_e2e.id}"
  depends_on          = ["ccloud_quota.kubernikus_e2e"]
}

resource "openstack_networking_router_interface_v2" "router_interface_e2e" {
  router_id = "${openstack_networking_router_v2.router_e2e.id}"
  subnet_id = "${openstack_networking_subnet_v2.subnet_e2e.id}"
}

data "openstack_networking_secgroup_v2" "kubernikus_e2e_default" {
  name        = "default"
  tenant_id   = "${openstack_identity_project_v3.kubernikus_e2e.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_e2e_ssh" {
  tenant_id = "${openstack_identity_project_v3.kubernikus_e2e.id}"
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "tcp"
  remote_ip_prefix  = "10.180.0.0/16"
  port_range_min    = 22
  port_range_max    = 22
  security_group_id = "${data.openstack_networking_secgroup_v2.kubernikus_e2e_default.id}"

  depends_on          = ["ccloud_quota.kubernikus_e2e"]
}
