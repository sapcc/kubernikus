TF_PLUGIN_DIR          ?= "$(GOPATH)/bin"

TF_AUTH_URL            ?= "https://identity-3.$(TF_REGION).cloud.sap/v3"
TF_USER_DOMAIN         := ccadmin
TF_DOMAIN_NAME         := ccadmin
TF_TENANT              := kubernikus

TF_BACKEND_AUTH_URL    ?= "$(TF_AUTH_URL)"
TF_BACKEND_REGION      ?= "$(TF_REGION)"
TF_BACKEND_USER        ?= "$(TF_USER)"
TF_BACKEND_USER_DOMAIN ?= "$(TF_USER_DOMAIN)"
TF_BACKEND_PASSWORD    ?= "$(TF_PASSWORD)"

.PHONY: all validate init plan apply

ifndef TF_REGION
$(error Set environment variable TF_REGION to continue)
endif

ifndef TF_USER
$(error Set environment variable TF_USER to continue. Requires ccadmin/cloud_admin permissions.)
endif

ifndef TF_PASSWORD
$(error Set environment variable TF_PASSWORD to continue)
endif


all: validate init plan apply

validate:
	env | grep "TF_\|OS_"

init: validate
	terraform init \
		-plugin-dir=$(TF_PLUGIN_DIR) \
		-backend-config="auth_url=$(TF_BACKEND_AUTH_URL)" \
		-backend-config="region_name=$(TF_BACKEND_REGION)" \
		-backend-config="user_name=$(TF_BACKEND_USER)" \
		-backend-config="domain_name=$(TF_BACKEND_USER_DOMAIN)" \
		-backend-config="password=$(TF_BACKEND_PASSWORD)"

plan: validate
	terraform plan \
		-var region=$(TF_REGION) \
		-var user_name=$(TF_USER) \
		-var user_domain_name=$(TF_USER_DOMAIN) \
		-var password=$(TF_PASSWORD) \
		-var domain_name=$(TF_DOMAIN_NAME) \
		-var tenant_name=$(TF_TENANT) \
		-parallelism=0

apply: validate
	terraform apply \
		-input=false \
		-auto-approve \
		-var region=$(TF_REGION) \
		-var user_name=$(TF_USER) \
		-var user_domain_name=$(TF_USER_DOMAIN) \
		-var password=$(TF_PASSWORD) \
		-var domain_name=$(TF_DOMAIN_NAME) \
		-var tenant_name=$(TF_TENANT) \
		-parallelism=0
