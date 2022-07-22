FLY ?= fly7 -t services

default: clean pipeline.yaml
	$(FLY) set-pipeline -p kubernikus -c pipeline.yaml

clean: FORCE
	if [ -e pipeline.yaml ]; then rm pipeline.yaml; fi

%.yaml: %.yaml.erb
	if [ -e $@ ]; then rm $@; fi
	erb $< > $@

FORCE:
