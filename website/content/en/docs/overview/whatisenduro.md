---
title: "What is Enduro?"
linkTitle: "What is Enduro?"
weight: 1
description: >
  And its use case.
---

Enduro is a new automation tool built by Artefactual to support
advanced Archivematica setups and integrations with other systems and
workflows. Specifically, it was built to meet the following
requirements:

* High level of automation
* High throughput
* Processing visibility
* Single user interface to manage multiple Archivematica pipelines
* Issue resolution / Error handling capabilities
* Fault tolerance
* Additional workflows: pre-ingest, post-ingest

## Use case

Enduro provides automated workflows to multi-pipeline Archivematica
deployments that are integrated with other systems. Enduro will
automatically:

* Perform pre-ingest tasks such as metadata verification
* Transfer SIPs into Archivematica for ingest
* Monitor the ingest process and create an alert if an error occurs 
* Log an activity when the AIP is stored
* Send relevant data to external systems from the AIP
* Clean up the Archivematica dashboard and remove the SIP from the
  watched transfer folder for all successful ingests
