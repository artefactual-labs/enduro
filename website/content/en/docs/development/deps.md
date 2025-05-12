---
title: "Software dependencies"
linkTitle: "Software dependencies"
weight: 3
description: >
  What are our software dependencies and how to update them.
---

## Go

The version of Go used by this project is specified in the `go.mod` file.

`.go-version` is used by the Netlify project.

## Go modules

Use `make deps` to list available updates.

When updating Goa, make sure `hack/make/dep_goa.mk` is using the same version.

## OpenAPI Generator

In Makefile (target `ui-client`), as a Docker container.

## UI

Use `npm run deps-minor` to install available updates.

## Website

The version of Hugo is in `netlify.toml` and `hack/make/dep_hugo.mk`.

Docsy is the Hugo theme (`website/themes/docsy`), installed as a Hugo module.

## Other tools

See `hack/make/dep_*.mk`.
