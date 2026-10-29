---
schema_version: 1
id: "iss-2610262038188161"
slug: "universal-patterns-call-capture-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 786 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:90"
remedy: "Edit 04-universal-patterns.md section 5: say these verbs write no report (rather than 'are read-only renders'); capture, intent, spec, identity and banlist still write records."
finding_ref: cfk-a815a4948e5a9509
---

Universal patterns call `capture`, `intent` and `spec` read-only renders; the verification matrix has `abcd capture` write an issue into the ledger.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:90` — "`lint` and its targets, `capture`, `intent`, `guard`, `--version`, `rules`, `spec`, `identity` and `banlist` are read-only renders"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:63` — "writes an issue into the open ledger with its frontmatter populated"

04-universal-patterns groups `capture`, `intent` and `spec` with `lint` as read-only renders that write nothing. The verification matrix's capture-write row has `abcd capture "<text>"` write an issue into the open ledger, and `spec close` and `intent` creation also write records. A verb that writes ledger records is not a read-only render.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 786.
