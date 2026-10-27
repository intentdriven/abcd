---
schema_version: 1
id: "iss-2610262035153070"
slug: "itd-67-keeps-a-hand-typed-version"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 558 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:56"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-b5469dc8bc1e7648
---

itd-67 keeps a hand-typed --version override for a release; the status-board chapter says the version is derived and never hand-authored

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:56` — "`--version <x.y.z>` remains the override / major-bump escape hatch, not the norm."
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:511` — "The version is **derived, never hand-authored**: it is read from the shipped build, not from a literal in the record"

itd-67 keeps --version < x.y.z> as the override and major-bump escape hatch, a version a person types. The status-board chapter says the version is derived and never hand-authored, read from the shipped build and not from a literal. A version a human can set by hand contradicts a version that is never hand-authored.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 558.
