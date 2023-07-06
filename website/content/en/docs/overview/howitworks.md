---
title: "How Enduro works"
linkTitle: "How Enduro works"
weight: 2
---

Generally speaking Enduro "watches" directories or object stores (like
Amazon's S3 or MinIO). When a file is copied into these watched
sources an event tells Enduro to start a **processing workflow** for
the file.

The processing workflow consists of many activities (see the
_activity_ definition in Temporal's [glossary]) which are orchestrated
by Enduro.

This is how the activity summary looks in the user interface when you
check the workflow of a successful transfer:

![Activity summary](/activity-summary.jpg)

1. **createPackageLocalActivity** creates the Enduro collection giving
   it an identifier and naming it after the original file.

1. **ParseNameLocalActivity** is an NHA specific activity which
   validates the name of the source file according to their
   rules. Enduro's default configuration makes this activity a
   [NOP](<https://en.wikipedia.org/wiki/NOP_(code)>).

1. **loadConfigLocalActivity** loads the pipeline
   configuration. Things like its transfer and processing directories,
   credentials for contacting it, how many concurrent transfers can
   handle (its capacity), etc.

1. **acquire-pipeline-activity** checks the capacity of the pipeline
   to decide if we can send a new one.

1. **setStatusInProgressLocalActivity** updates the collection marking
   it as "in progress".

1. **download-activity** copies the original file from the watched
   source into a temporary file in the pipeline's processing directory.

1. **bundle-activity** prepares the temporary file in the pipeline's
   processing directory: if it's a compressed file it extracts it and it
   creates the objects and metadata directory expected by
   Archivematica. Then it copies the prepared directory into the
   pipeline's transfer source location.

1. **transfer-activity** submits the transfer to Archivematica and
   stores its UUID.

1. **updatePackageLocalActivity** updates the Enduro collection with
   the transfer UUID.

1. **poll-transfer-activity** checks the transfer status (think of
   looking at the Transfer tab in the Archivematica dashboard)

1. **updatePackageLocalActivity** updates the Enduro collection with
   the SIP UUID once the transfer finishes.

1. **poll-ingest-activity** checks the SIP status (think of looking at
   the Ingest tab in the Archivematica dashboard)

1. **clean-up-activity** removes the directory that Enduro created in
   the pipeline's transfer source location.

1. **releasePipelineLocalActivity** updates the pipeline capacity
   saying we're done using it.

1. **hide-package-activity** is executed twice to hide the transfer
   from the Transfer and Ingest tabs in the Dashboard.

1. **delete-original-activity** schedules a time when the original
   file has to be removed from the watched source.

1. **updatePackageLocalActivity** updates the final status of the
   Enduro collection.

## Local and "non local" activities

Notice that some of these activities are marked as **(local
activity)** which means they're supposed to run in the same host that
is running the processing workflow (see the _local activity_
definition in Temporal's [glossary]).

The other type of activities (non local) can theoretically be run in a
separate host (which can provide better performance, more resilience,
etc), but in practice we're [running them all in the same host at the
moment][issue-37].

[glossary]: https://docs.temporal.io/glossary
[issue-37]: https://github.com/artefactual-labs/enduro/issues/37
