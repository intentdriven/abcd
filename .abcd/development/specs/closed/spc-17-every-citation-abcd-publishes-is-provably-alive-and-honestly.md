---
id: spc-17
slug: every-citation-abcd-publishes-is-provably-alive-and-honestly
intent: itd-101
---
# every-citation-abcd-publishes-is-provably-alive-and-honestly

## Summary

spc-17 delivers itd-101's citation gate: a deterministic, zero-network lint
family that runs in the commit gate, and an explicit on-demand refresh verb
(`abcd docs cite refresh`) that does the live fetching and writes a committed
baseline the lint then enforces offline. The design decisions below were
settled by the 2026-07-27 grill (recorded in the intent's Grill Settlements);
this spec records them together with the mechanism — it does not reopen them.

## Settled constraints (from the grill)

- **Zero network in the gate.** The citations lint family reads only committed
  config and the committed baseline. All fetching lives in the refresh verb.
- **Refresh is manual now.** It is surfaced by ahoy status and
  release-preflight nagging; a scheduled-CI wrapper is a later, separately
  signed-off change and is out of this spec's scope.
- **Staleness policy.** A baseline entry older than 180 days makes docs lint
  warn; one older than 365 days blocks at the release gate only — commits are
  never calendar-blocked. Human-verified entries age on the same clock and
  re-enter the manual queue when stale.
- **Honest labelling, never method.** Each baseline entry records whether
  verification was automatic or manual, and when — never how a human verified.
- **The row-has-footnote rule is homed here.** The structural rule deferred
  from spc-15 (every crosswalk table row carries at least one footnote) is
  implemented in this intent or not at all.

## Mechanism

### Offline lint family (commit gate)

The docs-lint family gains citation rules, evaluated with zero network:

- **Structure**: footnote markers and definitions in bijection per page;
  every crosswalk table row carries at least one footnote (the spc-15
  deferral).
- **Syntax**: cited URLs and DOIs are well-formed.
- **Source policy**: aggregator domains are refused, from committed config.
- **Baseline enforcement**: no cited URL absent from the baseline, no entry
  recorded as broken, no entry whose recorded final address has drifted from
  the cited URL's current resolution record, and the staleness clock above
  (180-day warn here; the 365-day blocker binds in the release gate, not the
  commit gate).

Severities mirror the existing docs-lint family (blocker/warn), and findings
render with the same successor-carrying shape.

### Committed baseline

`abcd docs cite refresh` fetches every cited URL and writes the baseline, a
committed machine record under `.abcd/` (one file, schema-versioned). Per URL
it records: the final resolved address after redirects, when it was last
checked, the outcome, and the verification label — `automatic`, or `manual`
with its date. Nothing else: no fetch transcript, no method, no headers.

### Manual queue, one receipt schema

Sources that block automated fetchers join the manual queue:

- **Rung 1 (this spec's floor)**: the refresh verb prints the queue as a
  checklist; the maintainer clears it link by link and a confirm verb writes
  the dated receipt into the baseline (verification: manual + date).
- **Rung 2 (same schema, later rung)**: a generated, disposable checklist
  page hands back a receipt file that the same confirm verb ingests. One
  receipt schema serves both rungs — the page is a different producer of the
  same input, not a second pathway.

### Engine and seam

The fetch engine is native and dependency-free (net/http, bounded redirects,
explicit timeouts). A specialist link checker can later slot in behind the
refresh seam as an adapter without changing gate semantics: the baseline
schema and the lint rules are the contract; the fetcher is a replaceable
producer.

### Surfacing

ahoy status reports the baseline's age summary (stale/overdue counts), and
the release preflight nags when entries approach or pass the 365-day blocker.
Both planes are wired at delivery: CLI verbs and the plugin markdown surface.

## Acceptance-criteria mapping

- AC 1 (zero-network gate: structure, syntax, policy from committed inputs)
  → Offline lint family.
- AC 2 (baseline records final address, checked-when, auto-vs-manual + date,
  never how) → Committed baseline.
- AC 3 (180-day warn in docs lint; 365-day release-gate blocker; manual
  verifications age the same) → Settled constraints + Offline lint family.
- AC 4 (printed checklist + confirm verb; generated page hands back a receipt
  the same verb ingests; one schema) → Manual queue, one receipt schema.
- AC 5 (specialist checker slots behind the seam without gate changes) →
  Engine and seam.

## Out of scope

- The scheduled-CI refresh wrapper (explicitly out of this intent, per the
  grill; needs its own sign-off).
- The generated checklist page itself may land as the later rung; only the
  receipt schema it must emit is fixed here.
- Any change to non-citation docs-lint rules.
