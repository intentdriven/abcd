---
schema_version: 1
id: "iss-2610262031575425"
slug: "itd-82-requires-a-remedy-on-every-new"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 200 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-82-drain-ledger-triage.md:44"
remedy: "In the capture acceptance criterion of the brief press release, change the command to `/abcd:capture \"<text>\" --remedy \"<fix>\"` and list remedy among the populated frontmatter."
related_intents: [itd-82]
finding_ref: cfk-cf2f0db47d0c60ba
---

itd-82 requires a remedy on every new issue; the brief's acceptance criterion has /abcd:capture "< text>" alone produce a structured issue.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:44` — "and requires it of every new issue (ruling BX3 of 2026-09-29)"
- End B: `.abcd/development/brief/01-product/01-press-release.md:43` — "the user runs `/abcd:capture "<text>"`, **when** the command completes"

itd-82 says capture gains the remedy field 'and requires it of every new issue (ruling BX3 of 2026-09-29)'. The brief press release's acceptance criterion begins: Given 'the user runs `/abcd:capture "<text>"`, **when** the command completes', then a structured issue exists in open/. No remedy is supplied in that criterion. Under itd-82's rule that invocation is refused, so the criterion cannot pass as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 200.
