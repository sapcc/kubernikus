TF_PLUGIN_DIR             ?= $(GOPATH)/bin
TF_VARS_DIR               ?= .

TF_AUTH_URL               ?= https://identity-3.$(TF_REGION).cloud.sap/v3
TF_USER                   ?= kubernikus-terraform
TF_USER_DOMAIN            ?= Default
TF_PROJECT_DOMAIN         := ccadmin
TF_PROJECT                := kubernikus

TF_BACKEND_AUTH_URL       ?= $(TF_AUTH_URL)
TF_BACKEND_REGION         ?= $(TF_REGION)
TF_BACKEND_USER           ?= $(TF_USER)
TF_BACKEND_USER_DOMAIN    ?= $(TF_USER_DOMAIN)
TF_BACKEND_PASSWORD       ?= $(TF_PASSWORD)

.PHONY: all validate init plan apply

ifndef TF_REGION
$(error Set environment variable TF_REGION to continue)
endif

ifndef TF_PASSWORD
$(error Set environment variable TF_PASSWORD to continue)
endif


all: validate init plan apply


init:
	@terraform init \
		-plugin-dir=$(TF_PLUGIN_DIR) \
		-backend-config="auth_url=$(TF_BACKEND_AUTH_URL)" \
		-backend-config="region_name=$(TF_BACKEND_REGION)" \
		-backend-config="user_name=$(TF_BACKEND_USER)" \
		-backend-config="user_domain_name=$(TF_BACKEND_USER_DOMAIN)" \
		-backend-config="password=$(TF_BACKEND_PASSWORD)"

plan:
	@terraform plan \
		-var region=$(TF_REGION) \
		-var user_name=$(TF_USER) \
		-var user_domain_name=$(TF_USER_DOMAIN) \
		-var password=$(TF_PASSWORD) \
		-var domain_name=$(TF_PROJECT_DOMAIN) \
		-var tenant_name=$(TF_PROJECT) \
		-var-file="${TF_VARS_DIR}/kubernikus.tfvars" \
		-parallelism=10

apply:
	@terraform apply \
		-input=false \
		-auto-approve \
		-var region=$(TF_REGION) \
		-var user_name=$(TF_USER) \
		-var user_domain_name=$(TF_USER_DOMAIN) \
		-var password=$(TF_PASSWORD) \
		-var domain_name=$(TF_PROJECT_DOMAIN) \
		-var tenant_name=$(TF_PROJECT) \
		-var-file="${TF_VARS_DIR}/kubernikus.tfvars" \
		-parallelism=10 \
