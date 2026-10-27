---
schema_version: 1
id: "iss-2610262034397453"
slug: "itd-200-puts-the-user-level-status-line"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 512 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109"
remedy: "none (filed automatically)"
related_intents: [itd-200]
finding_ref: cfk-8d8babc122f48c3b
---

itd-200 puts the user-level status-line defaults under ~/.abcd/; the library chapter names the user-level home ~/.abcd.noindex/.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109` — "The user-level defaults under `~/.abcd/`, the managed-only override"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:64` — "The corpus is the default one under the user-level home, `~/.abcd.noindex/sources`"

itd-200's scope places the status line's user-level defaults, the managed-only override and the off switch under ~/.abcd/. 07-library states that the default corpus sits under the user-level home, ~/.abcd.noindex/sources. Two directory names for the one user-level home mean a builder following itd-200 writes the status-line defaults to a directory the rest of the record does not read.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 512.
