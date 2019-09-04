---
title: "Contributing to These Docs"
linkTitle: "Contributing to These Docs"
weight: 2
description: >
  Help us write better documentation.
---

Our docs are served from: https://enduroproject.netlify.com/. This is kindly
offered by Netlify. We are not using a custom domain yet.

The docs are built with [Hugo][hugo]. The sources are hosted in the Enduro
repository: https://github.com/artefactual-labs/enduro/, under the
[website][website-sources] directory.

We're using the [Docsy][docsy] theme. Its website has relevant documentation on
how to add contents to a project like ours.

Submit a pull request when your changes are ready. You will see a number of
GitHub checks that need to pass. The `netlify/enduroproject/deploy-preview`
check includes a link with a live preview of the docs including your changes.

You can serve the docs locally which is the best option during your editing
workflow. With Hugo installed, change to the `website` directory and
run:

    hugo serve

The site is now available at http://localhost:1313. The site is updated
automatically when you make changes to the sources.

[netlify]: https://www.netlify.com/
[hugo]: https://gohugo.io/
[docsy]: https://www.docsy.dev
[website-sources]: https://github.com/artefactual-labs/enduro/tree/main/website
