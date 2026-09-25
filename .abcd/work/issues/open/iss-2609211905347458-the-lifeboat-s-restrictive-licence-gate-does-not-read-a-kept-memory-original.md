---
schema_version: 1
id: "iss-2609211905347458"
slug: "the-lifeboat-s-restrictive-licence-gate-does-not-read-a-kept-memory-original"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "product thinker interview closing itd-36 as delivered, 2026-09-21"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat (disembark gates); .abcd/memory/sources/"
deferred_after: "v0.10.0"
deferral_reason: "needs a product-thinker ruling on whether a lifeboat carries the project's memory at all: disembark's plan and pack never pack .abcd/memory today (internal/core/lifeboat/plan.go and pack.go name no memory path), so a licence gate on memory pages and kept originals would guard a payload that is never packed; ruling P in autonomous run A's rulings-owed list, 2026-09-25"
---

The lifeboat restrictive-licence gate does not read a kept memory original. itd-36 (closed as delivered on 2026-09-21) asked that disembark refuse to surface .abcd/memory/sources/<sha256>.<ext> (an original kept with --keep-original) unless the launch allowlist names it, and refuse a gated payload carrying a GPL-3.0 citation when the project publishes as MIT; the memory pages carry source.licence today, and disembark packs the record families by path without reading that field. Wanted: the disembark gate reads source.licence on memory pages and the kept originals, refuses a restrictive licence against the project licence, and names the allowlist that admits one.

Finding (autonomous run A, lifeboat lane review, 2026-09-25): disembark's plan and pack never pack `.abcd/memory`, so the gate this record asks for has nothing to guard today. Building it first needs a ruling on whether a lifeboat should carry the project's memory; if it should, the pack change and this gate land together.
