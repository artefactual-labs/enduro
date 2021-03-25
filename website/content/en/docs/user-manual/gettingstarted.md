---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
description: >
  Explore the Enduro dashboard and perform a batch import.
---

## The Enduro dashboard

This is the Enduro Collections dashboard, the default landing page for the application. The dashboard lists individual transfers that were part of batch ingests and shows their processing status.
* QUEUED: The transfer has been included in a batch import and is awaiting ingest into Archivematica.
* IN PROGRESS: The transfer has been ingested into Archivematica and is being processed.
* DONE: The transfer has been processed in Archivematica and the resulting AIP has been placed in archival storage.
* ERROR: The transfer was ingested into Archivematica but an error prevented it from being packaged into an AIP and/or
 placed in archival storage.

![Capture3](https://user-images.githubusercontent.com/14101311/112519120-48eb8d80-8d57-11eb-832a-be5db75cae44.PNG)

## Preparing transfers for batch import

* Enduro is designed to import and queue up multiple transfers for ingest into Archivematica. In order to prepare your holdings for import, you will need to place your transfers in a location that your administrator has set up for this purpose.
* Once your transfers are in place, you will be telling Enduro where to look for them to start the import process. You will be directing Enduro to look in a given parent directory, and when it does it will import all of the top-level subdirectories in that parent directory.
* For example, if you create a top-level directory called *Test*, and place subdirectories called *Nature*, *Household items* and *Buildings* inside it, Enduro will consider each of those subdirectories to be transfers (regardless of the directory structure within those subdirectories).

<img src="https://user-images.githubusercontent.com/14101311/112539210-f10c5100-8d6d-11eb-91ae-9b3afd8835e3.png" width=30% height=30%>

## Starting a batch import

* To start a new batch ingest, click on Batch import in the upper right corner. This will open a new batch import page.
* Enter the path of the directory containing the transfers to be ingested. 
* Enter the name of the Archivematica processing pipeline that will be used to perform the ingest. 
* Click Submit. Note that the page does not change when Submit is clicked - don’t re-click.

![Capture7](https://user-images.githubusercontent.com/14101311/112523923-80a90400-8d5c-11eb-96c2-dd9df3ce3ad9.PNG)

* Return to the Collections page. You will see a set of new transfers with a status of QUEUED or IN PROGRESS. 
* As each transfer is processed into an AIP and placed in archival storage, its status will change to DONE. Congratulations! You have just used Enduro to perform batch processing! 
