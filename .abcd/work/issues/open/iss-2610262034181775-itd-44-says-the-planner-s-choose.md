---
schema_version: 1
id: "iss-2610262034181775"
slug: "itd-44-says-the-planner-s-choose"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 486 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:142"
remedy: "Rewrite itd-44's 'Plan/reclassify refusal guard' against the shipped Go verbs `abcd intent plan` and `abcd intent reclassify`, which never read `suggested_kind`; drop `_choose_candidate_kind`, `plan_single` and `_PLAN_CANDIDATE_KINDS` (predecessor names absent from the tree) and say where a `decision` value would be refused."
related_intents: [itd-44]
finding_ref: cfk-441541c42aff8417
---

itd-44 says the planner's `_choose_candidate_kind` reads `suggested_kind`; the intent chapter says no shipped code reads that field and planning never consults it.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:142` — "`_choose_candidate_kind` reads `suggested_kind` but only routes values in"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:93` — "no shipped code reads the field at all. Planning never consults it, defaulting a null `kind` to `standalone`."

itd-44's guard against planning a `decision` draft assumes the planner reads `suggested_kind` and could plan such a draft as `standalone` through an interactive prompt. 05-intent.md says no shipped code reads the field, and planning sets a null kind to `standalone` without consulting it. The guard is designed against a code path that, according to the intent chapter, does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 486.
