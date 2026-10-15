---
id: dcn-2609010000000002
slug: the-changelog-composer-writes-only-added
date: 2026-09-01
---
2026-09-01 — The changelog composer writes only Added and Fixed until it can see the previous release (maintainer, ruling on iss-2609011207114761). The composer's inputs are the records that shipped and their bodies, never the base tag's surface, yet Changed, Deprecated and Removed are claims about exactly that surface, and in v0.7.0 it wrote three such lines about a verb that did not exist in v0.6.9. From here the composer emits no Changed, Deprecated or Removed section, the ingest refuses a composed changelog that carries one, and the release notes say so once so the absence reads as a rule rather than an oversight. Rejected: handing the composer the base tag's command reference (the right long-term input, deferred until the release surface is exported in a form the composer can diff); leaving it to the pre-tag review (a review is not a gate).
