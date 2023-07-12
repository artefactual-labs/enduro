$(call _assert_var,MAKEDIR)
$(call _conditional_include,$(MAKEDIR)/base.mk)
$(call _assert_var,UNAME_OS)
$(call _assert_var,UNAME_ARCH)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

GORELEASER_VERSION ?= 1.19.2

GORELEASER := $(CACHE_VERSIONS)/goreleaser/$(GORELEASER_VERSION)
$(GORELEASER):
	@rm -f $(CACHE_BIN)/goreleaser
	@mkdir -p $(CACHE_BIN)
	$(eval TMP := $(shell mktemp -d))
	@curl -sSL \
		"https://github.com/goreleaser/goreleaser/releases/download/v$(GORELEASER_VERSION)/goreleaser_$(UNAME_OS)_$(UNAME_ARCH).tar.gz" \
		| tar xz -C $(TMP)
	@mv $(TMP)/goreleaser $(CACHE_BIN)/
	@chmod +x $(CACHE_BIN)/goreleaser
	@rm -rf $(dir $(GORELEASER))
	@mkdir -p $(dir $(GORELEASER))
	@touch $(GORELEASER)
