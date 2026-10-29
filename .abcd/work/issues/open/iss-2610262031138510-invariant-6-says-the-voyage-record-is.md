---
schema_version: 1
id: "iss-2610262031138510"
slug: "invariant-6-says-the-voyage-record-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 101 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:19"
remedy: "In 04-universal-patterns.md section 6, replace 'Both are tracked under the visibility rule' with: the log tier sits in the gitignored local tier, and voyage is operator-level under abcd's home, outside every checkout and never committed (invariant 6, adr-35)."
finding_ref: cfk-06a792042d745545
---

Invariant 6 says the voyage record is never committed, but universal patterns says voyage is tracked under the visibility rule

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:19` — "keyed on the root-commit SHA like the history store — never committed, and never accumulated as stale snapshots"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:161` — "Both are tracked under the visibility rule; they answer different questions"

Invariant 6 says operations history lives at the operator level, `~/.abcd.noindex/voyage/<source-root-sha>/`, and is never committed. Universal patterns compares the log tier with `voyage/` and says both are tracked under the visibility rule. Under that rule the `.abcd/` namespace is committed in a private repository. The voyage store cannot be both outside every commit and tracked under the switch that decides what a repository commits.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 101.
