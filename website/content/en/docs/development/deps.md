---
title: "Software dependencies"
linkTitle: "Software dependencies"
weight: 3
description: >
  What are our software dependencies and how to update them.
---

## Go

The version of Go used by this projected is determined in [`.go-version`](https://github.com/artefactual-labs/enduro/blob/main/.go-version).

## Go modules

Use `make deps` to list available updates.

When updating Goa, make sure `hack/make/dep_goa.mk` is using the same version.

## OpenAPI Generator

In Makefile (target `ui-client`), as a Docker container.

## UI

Use `npm run deps-minor` to install available updates.

## Website

The version of Hugo is in `netlify.toml` and `hack/make/dep_hugo.mk`.

Docsy is the Hugo theme (`website/themes/docsy`), installed as a submodule.
We're still using v0.6.0 until we can address the update to Bootstrap 5.2.

## Other tools

See `hack/make/dep_*.mk`.
