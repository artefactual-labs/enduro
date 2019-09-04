---
title: "Releases"
linkTitle: "Releases"
weight: 3
description: >
  How do we release Enduro.
---

The release process is automated via GitHub Actions.\
See [`release.yml`][release-workflow] for more - we plan to develop it further.

The artifacts associated with a release are automatically published after the
git tag is submitted to the repository. For example:

    git tag v0.1.0-alpha.2
    git push --tags

The `release` workflow is started as soon as the tag makes it to GitHub. The
artifacts will start building right away and they will appear in the
[Releases page][release-github-page] as soon as they're ready.

The workflow runs are recorded by GitHub for auditing purposes, e.g. check out
the [workflow run][first-release-run] that made our very first release.

[release-workflow]: https://github.com/artefactual-labs/enduro/blob/main/.github/workflows/release.yml
[release-github-page]: https://github.com/artefactual-labs/enduro/releases
[first-release-run]: https://github.com/artefactual-labs/enduro/commit/0f17f7fa3beb88f7039cf9b6e1dc1a6f4af66f51/checks
