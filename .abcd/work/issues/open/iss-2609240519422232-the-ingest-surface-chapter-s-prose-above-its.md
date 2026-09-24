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
---

The ingest surface chapter's prose above its generated-appendix marker enumerates ingest sub-verbs and omits 'history ingest', the one v0.10.0 crosscheck finding inside a chapter where itd-147's rule (prose above the marker states no flag and no sub-verb) applies; itd-147 ac-6 is therefore not met at fa744b41 (finding x-049).
