---
title: "Configuration"
linkTitle: "Configuration"
weight: 2
description: >
  Configuration options available in Enduro.
---

## Configuration

Enduro is configured by setting the relevant options in the ``enduro.toml``
file which uses TOML, a well-known file format for configuration files.

The default search paths are `/etc/enduro.toml`, `$HOME/.config/enduro.toml`,
and the current directory. Additionally, users can indicate the configuration
file using the optional argument `--config=example.toml`.

**This is work in progress.**

## Sections

### `[pipeline]`

Used to define Archivematica pipelines. For example:

```toml
[[pipeline]]
name = "am"
baseURL = "http://127.0.0.1:62080"
user = "test"
key = "test"
transferDir = "~/.am/ss-location-data"
processingDir = ""
processingConfig = "automated"
transferLocationID = "88f6b517-c0cc-411b-8abf-79544ce96f54"
storageServiceURL = "http://test:test@127.0.0.1:62081"
capacity = 3
retryDeadline = "10m"
transferDeadline = "24h"
```

#### `name`

E.g.: `"am"`

#### `baseURL`

E.g.: `"http://127.0.0.1:62080"`

#### `user`

E.g.: `"test"`

#### `key`

E.g.: `"test"`

#### `transferDir`

E.g.: `"~/.am/ss-location-data"`

#### `processingDir`

E.g.: `""`

#### `processingConfig`

E.g.: `"automated"`

#### `transferLocationID`

E.g.: `"88f6b517-c0cc-411b-8abf-79544ce96f54"`

#### `storageServiceURL`

E.g.: `"http://test:test@127.0.0.1:62081"`

#### `capacity` (Integer)

E.g.: `3`.

#### `retryDeadline` (String)

E.g.: `"10m"`.

#### `transferDeadline` (String)

If present, transfers in this pipeline will be automatically abandoned when the
deadline is exceeded, causing the workflow to complete with an error. It yields
the pipeline slot.

The string should be constructed as a sequence of decimal numbers, each with
optional fraction and a unit suffix, such as "30m", "24h" or "2h30m".
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

## Configuration example

Source: https://github.com/artefactual-labs/enduro/blob/main/enduro.toml.

{{< config >}}
