$(call _assert_var,MAKEDIR)
$(call _conditional_include,$(MAKEDIR)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

GOMAJOR_VERSION ?= 0.9.5

GOMAJOR := $(CACHE_VERSIONS)/gomajor/$(GOMAJOR_VERSION)
$(GOMAJOR):
	@rm -f $(CACHE_BIN)/gomajor
	@mkdir -p $(CACHE_BIN)
	@env GOBIN=$(CACHE_BIN) go install github.com/icholy/gomajor@v$(GOMAJOR_VERSION)
	@chmod +x $(CACHE_BIN)/gomajor
	@rm -rf $(dir $(GOMAJOR))
	@mkdir -p $(dir $(GOMAJOR))
	@touch $(GOMAJOR)
