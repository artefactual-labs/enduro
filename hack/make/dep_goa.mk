$(call _assert_var,MAKEDIR)
$(call _conditional_include,$(MAKEDIR)/base.mk)
$(call _assert_var,UNAME_OS)
$(call _assert_var,UNAME_ARCH)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

# Keep in sync with the goa version in go.mod.
GOA_VERSION ?= 3.13.2

GOA := $(CACHE_VERSIONS)/goa/$(GOA_VERSION)
$(GOA):
	@rm -f $(CACHE_BIN)/goa
	@mkdir -p $(CACHE_BIN)
	@env GOBIN=$(CACHE_BIN) go install goa.design/goa/v3/cmd/goa@v$(GOA_VERSION)
	@chmod +x $(CACHE_BIN)/goa
	@rm -rf $(dir $(GOA))
	@mkdir -p $(dir $(GOA))
	@touch $(GOA)
