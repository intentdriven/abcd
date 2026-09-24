---
schema_version: 1
id: "iss-2608270645473170"
slug: "the-capture-scanner-redaction-over-masks-the-owner-segment-o"
severity: "nitpick"
category: "bug"
source: "agent-observation"
found_during: "capture-owner-overredaction-2026-08-27"
found_at: "internal/adapter/scanner"
---

The capture/scanner redaction over-masks the owner segment of a public 'owner/repo' string, replacing a legitimate public GitHub org/user name with [redacted-user] even when it is the very repository the record lives in. Observed 2026-08-27 when a captured issue body referencing this repo's own owner/repo slug had the owner masked. Not harmful (the slug still reads), but a false positive: the identity/path redaction treats an owner/repo owner as a sensitive identifier unconditionally. Consider exempting the repo's own owner (and public org names) from identity redaction, or narrowing the owner-segment match. Adjacent: iss-324 (scanner ispathsegmentbyte includes the slash in a third-party path).

## Evidence

- 2026-09-23, autonomous run A: the repository's own public organisation name was rewritten as `[redacted-user]` six times in one shipped intent's Audit Notes (itd-67), by `abcd intent audit ingest`. This time the path was the per-machine private-names layer, which carried the name because the list had been inherited from an earlier setup. The ingest applies that layer to the verifier's prose without asking whether a name is the repository's own public owner, so a committed record lost a public name with no warning; the six placeholders are still in itd-67. The exemption this record asks for (the repository's own owner) would cover both paths, and `abcd banlist add` warning on the repository's own owner would stop the second at its source.
