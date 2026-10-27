---
schema_version: 1
id: "iss-2610262031582038"
slug: "05-personas-requires-press-release"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 202 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-82-drain-ledger-triage.md:22"
remedy: "none (filed automatically)"
related_intents: [itd-82]
finding_ref: cfk-1e5ce701d8dd381d
---

05-personas requires press-release quotes from the persona registry; itd-82's quote is attributed to an unnamed product thinker.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:22` — "said a product thinker who ran the first ledger drains by hand"
- End B: `.abcd/development/brief/01-product/05-personas.md:3` — "Customer quotes use placeholder personas from `.abcd/development/personas.json`"

05-personas says 'Customer quotes use placeholder personas from `.abcd/development/personas.json`': the persona is chosen by role and given its registered name rather than generic phrasing. itd-82's press-release quote is attributed with 'said a product thinker who ran the first ledger drains by hand', which names no registered persona. The convention and the record disagree on how a quote is attributed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 202.
