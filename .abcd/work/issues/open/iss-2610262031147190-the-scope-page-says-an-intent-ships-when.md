---
schema_version: 1
id: "iss-2610262031147190"
slug: "the-scope-page-says-an-intent-ships-when"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 103 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md:40"
remedy: "Edit 04-scope.md line 40: closing the last open spec that names an intent is what ships it (`spec close`), and a `--remainder` close leaves it planned."
finding_ref: cfk-39a0b99324326451
refines: [iss-2609100508566552]
---

The scope page says an intent ships when its linked spec closes, but the spec verb ships it only when no open spec still names it

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/04-scope.md:40` — "It reaches `shipped/` when its linked spec is closed: closing the spec is what ships the intent, and there is no separate ship command."
- End B: `.abcd/development/brief/04-surfaces/README.md:275` — "closes a spec and ships its linked intent (`planned/` → `shipped/`) only when no open spec is left naming it"

The scope page says an intent reaches `shipped/` when its linked spec is closed, so closing the spec is what ships it. The surfaces index's spec verb says `spec close` ships the linked intent only when no open spec is left naming it, because an intent owns one or more specs, and a `--remainder` close leaves it planned. Under the 1:n relation, closing a linked spec does not by itself ship the intent.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 103.
