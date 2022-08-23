OUTPUT         := _output
OUTPUT_BASE    := $(GOPATH)/src
INPUT_BASE     := github.com/sapcc/kubernikus
API_BASE       := $(INPUT_BASE)/pkg/apis
GENERATED_BASE := $(INPUT_BASE)/pkg/generated
BIN            := $(OUTPUT)/bin

.PHONY: client-gen informer-gen lister-gen deepcopy-gen

client-gen: $(BIN)/client-gen
	@rm -rf ./pkg/generated/clientset
	@mkdir -p ./pkg/generated/clientset
	$(BIN)/client-gen \
	  --go-header-file /dev/null \
	  --output-base $(OUTPUT_BASE) \
	  --input-base $(API_BASE) \
	  --clientset-path $(GENERATED_BASE) \
	  --input kubernikus/v1 \
	  --clientset-name clientset 

informer-gen: $(BIN)/informer-gen
	@rm -rf ./pkg/generated/informers
	@mkdir -p ./pkg/generated/informers
	$(BIN)/informer-gen \
	  --go-header-file /dev/null \
	  --output-base                 $(OUTPUT_BASE) \
	  --input-dirs                  $(API_BASE)/kubernikus/v1  \
	  --output-package              $(GENERATED_BASE)/informers \
	  --listers-package             $(GENERATED_BASE)/listers   \
	  --internal-clientset-package  $(GENERATED_BASE)/clientset \
	  --versioned-clientset-package $(GENERATED_BASE)/clientset 

lister-gen: $(BIN)/lister-gen
	@rm -rf ./pkg/generated/listers
	@mkdir -p ./pkg/generated/listers
	$(BIN)/lister-gen \
	  --go-header-file /dev/null \
	  --output-base    $(OUTPUT_BASE) \
	  --input-dirs     $(API_BASE)/kubernikus/v1 \
	  --output-package $(GENERATED_BASE)/listers 

deepcopy-gen: $(BIN)/deepcopy-gen
	 find . -name zz_generated.deepcopy.go -delete
	${BIN}/deepcopy-gen \
	  --input-dirs $(API_BASE)/kubernikus/v1 --input-dirs $(INPUT_BASE)/pkg/api/models \
	  -O zz_generated.deepcopy \
	  --bounding-dirs $(INPUT_BASE) \
	  --output-base $(OUTPUT_BASE) \
	  --go-header-file /dev/null 


$(OUTPUT)/bin/%:
	@mkdir -p _output/bin
	GOBIN=$(PWD)/_output/bin go install k8s.io/code-generator/cmd/$*@kubernetes-1.21.14
