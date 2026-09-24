---
id: spc-2609221657168936
slug: a-repository-abcd-manages-reports-back-to-abcd-itself-one
intent: itd-2609221656361680
origin: researcher-authored
production_mode: hand-written
---
# a-repository-abcd-manages-reports-back-to-abcd-itself-one

## Summary

The design record for itd-2609221656361680: the report template, the verb, the machine-store inbox, the greeting and the fingerprinted promotion.

## Scope

1. **The template** (`internal/core/report`): schema version, kind (enhancement | defect), severity, category, abcd version, surface, remedy, evidence pointers, prose; `--template` writes the skeleton, the validator refuses a malformed block naming the field (criteria 1, 6).
2. **The verb** `abcd report [<file>]`: validates, stamps the sender's root-commit key and the received time, writes to `~/.abcd/inbox/` through the atomic writer, prints the path (criteria 1, 2).
3. **The greeting**: the session-start hook and the status board read the inbox's count and the distinct sender count; one line each, no detail (criterion 3).
4. **The reading**: `abcd inbox` and `abcd inbox show <id>`, read-only, newest first, sender named; an unknown schema version renders as unreadable with the version (criteria 4, 6).
5. **The promotion**: `abcd inbox promote <id>` composes a capture through capture's own core with `source: managed-repo`, `found_during` naming the key and a generic description, the report id as the evidence pointer, and the scanner on the text; the report is marked promoted and kept (criteria 5, 7).
6. **The name**: a test greps the promoted capture and every committed artefact for the sender's declared name (criterion 7).

## Out of scope

- Cross-account or cross-machine delivery; automatic filing; a reply channel.

## Approach

The inbox is a sixth machine-scoped store beside history, transcripts, worktrees, sources and labs, and is read by the same session-start path that already reports staged transcripts; the promotion reuses capture's core so a promoted report is an ordinary ledger record.

## Footprint

- packages: internal/core/report, internal/core/capture, internal/core/positioning, internal/surface/cli, hooks/
- tests: the template validator on good and bad blocks; the write path and its location; the greeting counts; the unknown version; the promotion's fingerprint and the name grep

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 report filed against the template | scope 1, 2 |
| 2 lands in the machine store only | scope 2 |
| 3 the greeting | scope 3 |
| 4 read-only rendering | scope 4 |
| 5 nothing files itself; promotion fingerprints | scope 5 |
| 6 unknown version listed | scope 1, 4 |
| 7 no name in anything committed | scope 5, 6 |
