---
schema_version: 1
id: "iss-2608311234549104"
slug: "the-cold-reading-ingest-verb-s-echo-is-a-fourth-copy-of-the"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "echo is termsafe.CleanProseLine under this package's cap, the declared canonical home."
impact: internal
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The cold-reading ingest verb's echo() is a fourth copy of the sanitise-and-cap primitive. termsafe.CleanProse/CleanProseLine declares itself the canonical home for the untrusted-prose cleaner every host-delegated ingest boundary needs before a model-supplied string is written into a durable record, and lifeboat, release and ideate all route through it. echo also omits the HTML-opener neutralisation that home performs.

## Grounds

- pursued: the finding is closed by a test that fails without the change; a later review or mutation run finding the same shape again would show this wrong
