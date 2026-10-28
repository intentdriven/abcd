---
schema_version: 1
id: "iss-2610262033189165"
slug: "itd-67-sends-change-narration-stripped"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 386 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:48"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-4836e7b576c3f652
---

itd-67 sends change-narration stripped from a doc into the changelog; 04-launch says rephrase the doc and never move the sentence into the changelog

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:48` — "change-narration that gate strips from a doc body is appended here"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:309` — "so the remedy is to rephrase the doc, not to move the sentence into the changelog"

itd-67 says change-narration that the itd-65 doc-history gate strips from a doc body is appended to the auto-recorded changelog. 04-launch says the changelog is derived from the records (adr-37), so the fix for a narration finding is to rephrase the doc, not to move the sentence into the changelog. The changelog cannot both take in stripped doc prose and refuse it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 386.
