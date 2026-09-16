---
schema_version: 1
id: "iss-322"
slug: "lifeboat-gvidnum-discards-atoi-range-error-so-an-over-int64"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/lifeboat/graveyard_abandoned.go"
wontfix_reason: "self-verdicted cosmetic in the review tooling's own report rendering; MaxInt64 sorts last and findings are capped, so nothing actionable"
---

lifeboat gvIDNum discards Atoi range error so an over-int64 record id from a foreign repo sorts to the extreme of the graveyard readout
## Evidence
`internal/core/lifeboat/graveyard_abandoned.go:382-400` — `n, _ := strconv.Atoi(id[start:end])` over record ids from a foreign embarked/probed repo (validated only by `^adr-[0-9]+$` etc., digit-unbounded). `adr-99999999999999999999-x` clamps to MaxInt64. Same class as iss-309 (semver) and the fixed spec.go. <!-- record-lint: illustrative -->

## Adversarial verdict: CONFIRMED but cosmetic (nitpick) — RECORD-ONLY
gvIDNum's only caller is the `gvSortByID` comparator; `n` never flows to an index/allocation/`+1`, so there is no wrap-to-negative analogue. Ascending sort puts MaxInt64 LAST (not "ahead of all" as first stated); findings are capped at 500/signal. Worst case: one pathological record at the wrong end of a list — cosmetic, acceptable degradation over a foreign repo under the trusted-worktree model. Not fixed this round; if a shared "parse-ordinal-or-sentinel" helper is later hoisted across spec/launch/lifeboat, this rides along.
