$(call _assert_var,MAKEDIR)
$(call _conditional_include,$(MAKEDIR)/base.mk)
$(call _assert_var,UNAME_OS2)
$(call _assert_var,UNAME_ARCH2)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

HUGO_VERSION ?= 0.113.0

HUGO := $(CACHE_VERSIONS)/hugo/$(HUGO_VERSION)
$(HUGO):
	@rm -f $(CACHE_BIN)/hugo
	@mkdir -p $(CACHE_BIN)
	$(eval TMP := $(shell mktemp -d))
	@echo "https://github.com/gohugoio/hugo/releases/download/v$(HUGO_VERSION)/hugo_extended_$(HUGO_VERSION)_$(UNAME_OS2)_$(UNAME_ARCH2).tar.gz"
	@curl -sSL \
		"https://github.com/gohugoio/hugo/releases/download/v$(HUGO_VERSION)/hugo_extended_$(HUGO_VERSION)_$(UNAME_OS2)_$(UNAME_ARCH2).tar.gz" \
		| tar xz -C $(TMP)
	@mv $(TMP)/hugo $(CACHE_BIN)/
	@chmod +x $(CACHE_BIN)/hugo
	@rm -rf $(dir $(HUGO))
	@mkdir -p $(dir $(HUGO))
	@touch $(HUGO)
