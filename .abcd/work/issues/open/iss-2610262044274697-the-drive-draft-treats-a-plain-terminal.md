---
schema_version: 1
id: "iss-2610262044274697"
slug: "the-drive-draft-treats-a-plain-terminal"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1208 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-2610222215485265-abcd-drain-run-from-a-plain-terminal.md:21"
remedy: "none (filed automatically)"
related_intents: [itd-2610222215485265, itd-2610161016401115]
finding_ref: cfk-677d290a6b13bca7
---

The drive draft treats a plain-terminal drain that runs on its own as new, but the shipped drain bundle says drains already run unattended in a plain terminal

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-2610222215485265-abcd-drain-run-from-a-plain-terminal.md:21` — "Type `abcd drain --drive` in a plain terminal and it works through the ledger on its own."
- End B: `.abcd/development/intents/shipped/itd-2610161016401115-a-drain-runs-unattended-on-a-hosted.md:21` — "A drain runs unattended on a hosted session or a plain terminal for a day."

itd-2610222215485265 introduces `--drive` as the only way for `abcd drain` to work through the ledger on its own from a plain terminal, and carries a proposed reversal of the host-delegated boundary to allow it. itd-2610161016401115, which the draft refines, already claims a drain runs unattended on a hosted session or a plain terminal for a day. Either the shipped claim overstates what exists, or the draft's premise is wrong.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1208.
