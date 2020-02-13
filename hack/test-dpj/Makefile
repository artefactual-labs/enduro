SHELL=/bin/bash

UUID := $(shell uuidgen | tr "[:upper:]" "[:lower:]")
TMPDIR := $(shell mktemp -d)
NAME := DPJ-SIP-$(UUID)

push-tar: content-creation archive-tar
	@echo "--> Uploading blob: enduro/sips/$(NAME).tar"
	@cat archive.tar | mc pipe enduro/sips/$(NAME).tar

archive-tar:
	@echo "--> Creating blob: $(CURDIR)/archive.tar"
	@cd $(TMPDIR) && tar cf $(CURDIR)/archive.tar .

push-tar-gz: content-creation archive-tar-gz
	@echo "--> Uploading blob: enduro/sips/$(NAME).tar.gz"
	@cat archive.tar.gz | mc pipe enduro/sips/$(NAME).tar.gz

archive-tar-gz:
	@echo "--> Creating blob: $(CURDIR)/archive.tar.gz"
	@cd $(TMPDIR) && tar czf $(CURDIR)/archive.tar.gz .

FIXITY_ERR ?= no
content-creation:
	@echo "--> Creating package: $(TMPDIR)/$(UUID)"
	@mkdir -p $(TMPDIR)/$(UUID)
	@rsync --exclude=".gitkeep" -aq $(CURDIR)/contents/ $(TMPDIR)/$(UUID)/
	@if [ "$(FIXITY_ERR)" = "yes" ]; then \
		mkdir -p $(TMPDIR)/metadata; \
		echo "2eb15cb1834214b05d0083c691f9545f  $(UUID)/DPJ-SIP/Journal/avlxml.xml" > $(TMPDIR)/metadata/checksum.md5; \
    fi

err: FIXITY_ERR = yes
err: push-tar

TIMES ?= 10
many-ok:
	@for run in {1..$(TIMES)}; do \
        make push-tar; \
    done;

TIMES ?= 10
many-err:
	@for run in {1..$(TIMES)}; do \
        make push-tar FIXITY_ERR=yes; \
    done;

TIMES ?= 10
many:
	@for run in {1..$(TIMES)}; do \
        fail=$$(shuf -e yes no -n 1); \
        make push-tar FIXITY_ERR=$${fail}; \
    done;
