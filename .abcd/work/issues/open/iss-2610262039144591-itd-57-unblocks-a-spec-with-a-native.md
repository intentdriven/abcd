---
schema_version: 1
id: "iss-2610262039144591"
slug: "itd-57-unblocks-a-spec-with-a-native"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 847 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-57]
finding_ref: cfk-6da74bff74e78fc3
---

itd-57 unblocks a spec with a native `spec rm-dep`, but the spec verb registers only its bare board and `spec close`

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:32` — "Unblock = native `spec rm-dep <spec> <sentinel>`"
- End B: `.abcd/development/brief/04-surfaces/README.md:275` — "The native spec store: bare invocation is a read-only status board, and `spec close` closes a spec"

itd-57 relies on an existing native dependency mechanism: a hold is a `depends_on` on the sentinel, a native `spec rm-dep` lifts it, and no new verb is in scope. The surfaces register describes the native spec verb as a read-only status board plus `spec close`, and nothing else. The block and unblock levers itd-57 calls native are not on the spec surface the brief registers.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 847.
