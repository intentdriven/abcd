---
schema_version: 1
id: "iss-2610262032042626"
slug: "the-context-page-says-abcd-intent-new-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 218 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md:11"
remedy: "none (filed automatically)"
related_intents: [itd-4]
finding_ref: cfk-38b9d1e2223d37be
---

The context page says `abcd intent new` is refused; itd-4's `capture promote` calls `/abcd:intent new`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/02-context.md:11` — "`new` is not a sub-verb: `abcd intent new` is refused and names the quoted create."
- End B: `.abcd/development/intents/shipped/itd-4-issue-capture.md:35` — "`capture promote <iss-N>` — promote an `iss-N` entry to an intent draft (calls `/abcd:intent new` with the entry body + bidirectional link)"

The brief says `new` is not a sub-verb and that `abcd intent new` is refused, pointing at the quoted create. itd-4 defines the promote path as a call to `/abcd:intent new`. A promote built from itd-4 calls a form the surface refuses.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 218.
