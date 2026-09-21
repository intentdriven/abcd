---
id: spc-2609211930059886
slug: review-queue-auto-drain-fidelity-gate
intent: itd-53
origin: researcher-authored
production_mode: hand-written
---
# review-queue-auto-drain-fidelity-gate

## Summary

The design record for itd-53 as re-scoped on 2026-09-21: `abcd intent audit
--owed`, one bounded command that runs the owed fidelity audits oldest
first through the host pass a single audit uses.

## Scope

1. **The listing** comes from itd-2609150819445595's owed-intents read
   (receipt ids per shipped intent); `--owed` orders it oldest first and
   applies `--max <n>` (criterion 1).
2. **Each audit**: `intent audit <itd-N>` writes the request; the plugin
   page hands it to the auditor agent one at a time; `intent audit ingest`
   applies the verdict; the loop continues with the next (criteria 1, 3).
3. **No reviewer**: the page detects no auditor available (the host's agent
   listing, or a refused launch) and the command leaves every entry owed
   with the reason in its summary (criterion 2).
4. **Not-met on a long-shipped intent** is captured through `capture` with
   the receipt named, never fixed by this command (criterion 3).
5. **The close hook** is untouched (criterion 4).

## Out of scope

- A boundary-hooked autodrain, a blocking gate, a scheduled run.
- Fixing anything: the audit-driven fix round is itd-50's, inside `build`.

## Approach

A thin loop in the plugin page over the existing request/ingest pair, with
the ordering and the cap computed by `internal/core/intent` from the owed
list; the CLI form prints the ordered list and the next request path, so a
host without the page can drive it by hand.

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 oldest first, capped, ingested | scope 1, 2 |
| 2 left owed when no reviewer | scope 3 |
| 3 lands as a single audit; not-met captured | scope 2, 4 |
| 4 close hook enqueue-only | scope 5 |
