---
schema_version: 1
id: "iss-2609240519422232"
slug: "the-ingest-surface-chapter-s-prose-above-its"
severity: "minor"
category: "drift"
source: "drift-detection"
found_during: "v0.10.0 release gate: brief-surface crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/14-ingest.md"
resolution: "The ingest chapter's prose above its appendix marker names no ingest path of another verb; it defers the list to the generated CLI reference, as adr-2609231028044006 requires."
impact: internal
resolved_by:
  commit: "7dafb541cffece145fe3a0c172b64672cd9f596f"
---

The ingest surface chapter's prose above its generated-appendix marker enumerates ingest sub-verbs and omits 'history ingest', the one v0.10.0 crosscheck finding inside a chapter where itd-147's rule (prose above the marker states no flag and no sub-verb) applies; itd-147 ac-6 is therefore not met at fa744b41 (finding x-049).

## Grounds

- pursued: prose above the marker that names no sub-verb cannot omit one; a later crosscheck finding a false-claim or stale-count about a flag or sub-verb in 14-ingest.md above the marker would show it wrong
