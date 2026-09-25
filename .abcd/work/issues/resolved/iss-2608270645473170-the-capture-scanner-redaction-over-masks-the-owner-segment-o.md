---
schema_version: 1
id: "iss-2608270645473170"
slug: "the-capture-scanner-redaction-over-masks-the-owner-segment-o"
severity: "nitpick"
category: "bug"
source: "agent-observation"
found_during: "capture-owner-overredaction-2026-08-27"
found_at: "internal/adapter/scanner"
resolution: "github_username no longer reports the remote's owner where it is the owner half of the repository's own owner/repo slug (ProbeIdentity reads the repository name too); the owner alone or as another repository's owner is still masked. The placeholder in the itd-67 evidence is the scanner's github_username placeholder, which only the scanner writes, so this exemption covers that path; the placeholders already in itd-67 are left as committed."
impact: fix
resolved_by:
  commit: "e55cefd2055bcbd5ca94a110f652e7a034ff753f"
---

The capture/scanner redaction over-masks the owner segment of a public 'owner/repo' string, replacing a legitimate public GitHub org/user name with [redacted-user] even when it is the very repository the record lives in. Observed 2026-08-27 when a captured issue body referencing this repo's own owner/repo slug had the owner masked. Not harmful (the slug still reads), but a false positive: the identity/path redaction treats an owner/repo owner as a sensitive identifier unconditionally. Consider exempting the repo's own owner (and public org names) from identity redaction, or narrowing the owner-segment match. Adjacent: iss-324 (scanner ispathsegmentbyte includes the slash in a third-party path).

## Evidence

- 2026-09-23, autonomous run A: the repository's own public organisation name was rewritten as `[redacted-user]` six times in one shipped intent's Audit Notes (itd-67), by `abcd intent audit ingest`. This time the path was the per-machine private-names layer, which carried the name because the list had been inherited from an earlier setup. The ingest applies that layer to the verifier's prose without asking whether a name is the repository's own public owner, so a committed record lost a public name with no warning; the six placeholders are still in itd-67. The exemption this record asks for (the repository's own owner) would cover both paths, and `abcd banlist add` warning on the repository's own owner would stop the second at its source.

## Grounds

- pursued: the repository's own slug is written as authored by every scanner-backed redactor; a capture or ingest that still masks the owner in that slug would show it wrong
