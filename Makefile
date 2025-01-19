BUILD_DIR=build
VYOSDIR=vyos-1x

%.xml: %.xml.in
	@echo Generating $(VYOSDIR)/$(BUILD_DIR)/$@ from $<
	mkdir -p $(VYOSDIR)/$(BUILD_DIR)/interface-definitions
	$(VYOSDIR)/scripts/transclude-template $< > $(VYOSDIR)/$(BUILD_DIR)/interface-definitions/$(basename $(notdir $@)).xml

config_xml_src = $(wildcard ${VYOSDIR}/interface-definitions/*.xml.in)
config_xml_obj = $(config_xml_src:.xml.in=.xml)

.PHONY: interface-definitions all
all: vyos.json

interface-definitions: $(config_xml_obj)

vyos.json: interface-definitions vyos-to-json
	./vyos-to-json --interface-definitions ${VYOSDIR}/${BUILD_DIR}/interface-definitions --out vyos.json

vyos-to-json: vyos-to-json.go
	go build vyos-to-json.go
