BUILD_DIR=build
VYOSDIR=vyos-1x

%.xml: %.xml.in
	@echo Generating $(VYOSDIR)/$(BUILD_DIR)/$@ from $<
	mkdir -p $(VYOSDIR)/$(BUILD_DIR)/interface-definitions
	$(VYOSDIR)/scripts/transclude-template $< > $(VYOSDIR)/$(BUILD_DIR)/interface-definitions/$(basename $(notdir $@)).xml

config_xml_src = $(wildcard ${VYOSDIR}/interface-definitions/*.xml.in)
config_xml_obj = $(config_xml_src:.xml.in=.xml)

.PHONY: interface-definitions all
all: vyos.json.gz

interface-definitions: $(config_xml_obj)

vyos.json: interface-definitions vyos-to-json
	./vyos-to-json --interface-definitions ${VYOSDIR}/${BUILD_DIR}/interface-definitions --out vyos.json

vyos-to-json: vyos-to-json.go
	go build vyos-to-json.go

# gzip the newly generated vyos.json and then copy that into into
# syntax/, using the date of the most recent VyOS commit as part of
# the name.
vyos.json.gz: vyos.json
	gzip -9c vyos.json > vyos.json.gz
	cp vyos.json.gz syntax/vyos-$(shell date -d "$(cd vyos-1x ; git log -n 1 --date-order | grep Date: | cut -d: -f2- | cut -c4- | cut -d+ -f1)"  +%Y%m%d).json.gz

