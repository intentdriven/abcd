---
schema_version: 1
id: "iss-2610262035583798"
slug: "the-verification-matrix-says-an-imported"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 612 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:32"
remedy: "none (filed automatically)"
finding_ref: cfk-9e6f404c12389b1c
---

The verification matrix says an imported transcript merges by timestamp and content hash; the history chapter keys identity on content hash, session id and source labels, so the same content under another session is a new record

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:32` — "an optional import merges by timestamp and content hash over the same store"
- End B: `.abcd/development/brief/04-surfaces/11-history.md:58` — "the same content under a different session id writes a new record"

The matrix's native-transcript-capture row says an optional import merges by timestamp and content hash over the same store. The history chapter says capture, the path an import takes under the `import` route, is idempotent on the content hash, session id and source labels together, and that the same content under a different session id writes a new record. A merge keyed on timestamp and hash would fold together two sessions' identical content, which the history chapter forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 612.
