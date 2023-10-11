$(call _assert_var,MAKEDIR)
$(call _conditional_include,$(MAKEDIR)/base.mk)
$(call _assert_var,UNAME_OS2)
$(call _assert_var,UNAME_ARCH2)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

TEMPORAL_CLI_VERSION ?= 0.10.6

TEMPORAL_CLI := $(CACHE_VERSIONS)/temporal/$(TEMPORAL_CLI_VERSION)
$(TEMPORAL_CLI):
	@rm -f $(CACHE_BIN)/temporal
	@mkdir -p $(CACHE_BIN)
	$(eval TMP := $(shell mktemp -d))
	@curl -sSL \
		"https://temporal.download/cli/archive/v$(TEMPORAL_CLI_VERSION)?platform=$(UNAME_OS2)&arch=$(UNAME_ARCH2)" \
		| tar xz -C $(TMP)
	@mv $(TMP)/temporal $(CACHE_BIN)/
	@chmod +x $(CACHE_BIN)/temporal
	@rm -rf $(dir $(TEMPORAL_CLI))
	@mkdir -p $(dir $(TEMPORAL_CLI))
	@touch $(TEMPORAL_CLI)
