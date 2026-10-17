---
id: dcn-2608060000000003
slug: iss-187-bug-hunt-loop-round-1-rules
date: 2026-08-06
---
2026-08-06 — iss-187 (bug-hunt loop, round 1): `rules.Merge`
(`internal/core/rules/rules.go`) assigned override domain keys straight into
`out.Domains` without ever allocating that map, because its `cloneRuleSet`
helper deliberately preserves a nil `Domains` (it only allocates when the
source map is non-nil). A base such as `RuleSet{SchemaVersion: 1}` — which
`Validate` accepts — therefore panicked with "assignment to entry in nil map"
the moment the overlay carried a domain, contradicting `Merge`'s own doc
comment ("New domain keys are added"), which states no such precondition. Not
reachable in production today: the sole call site, `RuleSet.Load`, always
passes `Merge(Defaults(), over)` and `Defaults()` always has a populated map —
so this was a latent defect in an exported API contract that would go live for
the first caller merging onto a non-`Defaults()` base (a future multi-tier
overlay starting from an empty set). Fixed at the `Merge` call site by
allocating `out.Domains` when it is nil and the overlay has at least one key,
matching the idiom the sibling loader `guard.Merge`
(`internal/core/guard/config.go`) already uses for `out.Entries`; `cloneRuleSet`
keeps its nil-preserving semantics, so no other caller's behaviour moves.
Repro: `internal/core/rules/rules_test.go`,
`TestMergeNilBaseDomainsAddsNewKeys`, watched failing on pre-fix code for the
claimed reason (panic: assignment to entry in nil map) and passing after.
