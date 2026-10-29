---
schema_version: 1
id: "iss-2610262037558450"
slug: "the-load-check-intent-keeps-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 759 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/27-implement.md:253"
remedy: "none (filed automatically)"
related_intents: [itd-2609231434459890]
finding_ref: cfk-4be06a6978dc06ba
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The intent shipped before the user scope was renamed; the shipped rename intent and its ADR say which name holds, and the code (machineload/limits.go) reads the .abcd.noindex path."
---

The load-check intent keeps the per-machine limits under ~/.abcd/; the implement chapter keeps them at ~/.abcd.noindex/load-limits.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/27-implement.md:253` — "The limits live in the caller's machine tier, `~/.abcd.noindex/load-limits`"
- End B: `.abcd/development/intents/shipped/itd-2609231434459890-abcd-s-own-test-lanes-check-the-machine.md:78` — "Per-machine limits in a settings file under `~/.abcd/`"

itd-2609231434459890, in its scope and its decision 5, puts the per-machine limits in a settings file under ~/.abcd/. 27-implement.md reads them from ~/.abcd.noindex/load-limits. The one machine tier carries two names. Since itd-2610030720038073 renamed the home and has an existing ~/.abcd stop every abcd command, a limits file placed where the intent says is never read.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 759.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The intent shipped before the user scope was renamed; the shipped rename intent and its ADR say which name holds, and the code (machineload/limits.go) reads the .abcd.noindex path.
