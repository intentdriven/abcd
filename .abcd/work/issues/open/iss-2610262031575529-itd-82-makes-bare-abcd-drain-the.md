---
schema_version: 1
id: "iss-2610262031575529"
slug: "itd-82-makes-bare-abcd-drain-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 201 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-82-drain-ledger-triage.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-82]
finding_ref: cfk-1f594b9998f64c44
---

itd-82 makes bare abcd drain the unattended run that changes state; 04-naming rules that a bare invocation renders and that changing state is what a sub-verb is for.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:20` — "`abcd drain` works the open issue ledger unattended"
- End B: `.abcd/development/brief/02-constraints/04-naming.md:92` — "Every `/abcd:<verb>` treats the bare invocation as status plus help plus a render"

itd-82's press release opens with '`abcd drain` works the open issue ledger unattended': the bare verb opens lanes and pull requests and writes markers, and decision 3 makes it 'all by default, capped by flag'. 04-naming states 'Every `/abcd:<verb>` treats the bare invocation as status plus help plus a render' and that a sub-verb earns its existence by mutating state. The planned verb builds in the bare-acts shape the naming rule excludes at design time; 04-naming treats acting verbs only as a gap in the shipped surface, not as an allowed design.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 201.
