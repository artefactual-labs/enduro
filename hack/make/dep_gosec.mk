$(call _assert_var,MAKEDIR)
$(call _conditional_include,$(MAKEDIR)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

GOSEC := $(CACHE_VERSIONS)/gosec/latest
.PHONY: $(GOSEC) # Ignored cached version, always download the latest.
$(GOSEC):
	rm -f $(CACHE_BIN)/gosec
	mkdir -p $(CACHE_BIN)
	echo Downloading github.com/securego/gosec/v2/cmd/gosec@latest
	env GOBIN=$(CACHE_BIN) go install github.com/securego/gosec/v2/cmd/gosec@latest
	rm -rf $(dir $(GOSEC))
	mkdir -p $(dir $(GOSEC))
	touch $(GOSEC)
