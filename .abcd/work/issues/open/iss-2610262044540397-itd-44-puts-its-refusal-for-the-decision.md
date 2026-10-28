---
schema_version: 1
id: "iss-2610262044540397"
slug: "itd-44-puts-its-refusal-for-the-decision"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1238 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:142"
remedy: "none (filed automatically)"
related_intents: [itd-44, itd-80]
finding_ref: cfk-eb2122e84575f3b2
---

itd-44 puts its refusal for the decision kind into a Python plan path (`_choose_candidate_kind`, `plan_single`); in shipped itd-80, planning is the native `abcd intent plan` verb

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:142` — "`_choose_candidate_kind` reads `suggested_kind` but only routes values in"
- End B: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:32` — "`abcd intent plan <itd-N>` mints a spec, writes the bidirectional link, sets `kind` (default `standalone`)"

itd-44 says the plan path reads `suggested_kind` through `_choose_candidate_kind` and routes only `_PLAN_CANDIDATE_KINDS`. Anything else falls through to an interactive prompt. itd-44 adds its refusal to `plan_single`/`reclassify`. itd-80 says the shipped `abcd intent plan <itd-N>` mints a native spec, writes the link, and sets `kind` with `standalone` as the default. The two records describe different plan paths, and itd-44's guard targets one the shipped verb does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1238.
