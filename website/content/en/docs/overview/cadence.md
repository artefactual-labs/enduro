---
title: "Cadence"
linkTitle: "Cadence"
weight: 3
description: >
  How we use Cadence to build Enduro.
---

Enduro is based on [Cadence][cadence-website], an [open-source][cadence-repo]
orchestration engine developed at Uber Engineering and led by the designers of
off-premises solutions like [Amazon Simple Workflow Service (SWF)][aws-swf] and
[Azure Durable Functions][azure-durable-functions].

Cadence is being used by some well-known companies, such as: Uber, HashiCorp,
Coinbase, LinkedIn or Banzai Cloud:

* Banzai Cloud shared a [series of blog posts][banzai-cloud-posts] on how they
  are using Cadence to implement complex workflows that manage the entire
  life-cycle of Kubernetes clusters.
* HashiCorp has recently shared details of [HashiCorp Consul Service][hcs], a
  managed Consul service built in partnership with Microsoft. Their control
  plane uses Cadence - see [the presentation][hcs-presentation] for more
  details.

[cadence-website]: https://cadenceworkflow.io/
[cadence-repo]: https://github.com/uber/cadence
[banzai-cloud-posts]: https://banzaicloud.com/tags/cadence/
[aws-swf]: https://aws.amazon.com/swf/
[azure-durable-functions]: https://docs.microsoft.com/en-us/azure/azure-functions/durable/durable-functions-overview
[hcs]: https://www.hashicorp.com/resources/making-multi-environment-service-networking-on-microsoft-azure-easy-with-consul
[hcs-presentation]: https://youtu.be/kDlrM6sgk2k?t=970
