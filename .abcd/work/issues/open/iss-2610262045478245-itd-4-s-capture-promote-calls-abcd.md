---
schema_version: 1
id: "iss-2610262045478245"
slug: "itd-4-s-capture-promote-calls-abcd"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1298 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-4-issue-capture.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-4, itd-2609212130136102]
finding_ref: cfk-86345cf5b22cc471
---

itd-4's capture promote calls /abcd:intent new; itd-2609212130136102 removes intent new

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-4-issue-capture.md:35` — "(calls `/abcd:intent new` with the entry body + bidirectional link)"
- End B: `.abcd/development/intents/shipped/itd-2609212130136102-abcd-s-verbs-consolidate-ahoy-s-three.md:40` — "`intent new` is removed"

itd-4 defines capture promote as calling /abcd:intent new with the entry body. itd-2609212130136102 removes intent new outright, so the promote path as itd-4 states it calls a verb that no longer exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1298.
