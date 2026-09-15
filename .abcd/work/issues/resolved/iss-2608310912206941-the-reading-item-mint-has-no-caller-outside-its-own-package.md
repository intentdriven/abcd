---
schema_version: 1
id: "iss-2608310912206941"
slug: "the-reading-item-mint-has-no-caller-outside-its-own-package"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-fidelity-audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/reading.go"
resolution: "The cold-reading ingest verb is that caller: reading.Ingest validates the reading's output and hands the items to capture.IngestReading, reachable from abcd reading ingest on the CLI and from the plugin markdown surface."
impact: additive
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

the reading item mint has no caller outside its own package so the admission and surprise schemas gate a corpus nothing can populate

Found by the itd-189 fidelity audit and verified independently by the
orchestrator. This is a "wired or it isn't done" finding.

`capture.IngestReading` mints `rdi-N`. Grepping the whole tree for callers finds
only its own test file: no CLI verb, no plugin surface, nothing under `cmd/`.
The CLI has verbs that CONSUME a reading item -- `capture promote rdi-N` and
`capture disposition rdi-N` -- and nothing that creates one.

So itd-189's admission and surprise schemas, and the outstanding-readings report
that reads them, are armed over a corpus the shipped product cannot populate.
Every acceptance criterion's Given clause presupposes a reading item that no
shipped surface can produce.

The producer is itd-185, the ingest door, which is `planned/` and unstarted.
Under itd-192 that makes the criteria MET_WITH_CONCERNS rather than NOT_MET,
because this phase does wire it -- and the auditor named itd-185 correctly.

**This materially widens iss-2608301726130926, the armed Phase-4 conditional.**
That record currently says two criteria flip to NOT_MET if itd-185 does not
land: itd-178 ac-2 and itd-180 ac-1. Itd-189's three criteria now rest on the
same intent. The conditional's blast radius is at least five criteria across
three intents, not two across two.

## Grounds

- pursued: the record writer gains its intended caller and the reading-record family becomes populatable from the shipped product; a run that ingested and left no rdi-N record would show this wrong
