---
schema_version: 1
id: "iss-2611040555430199"
slug: "the-scribe-chapter-says-ingest-proves"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
remedy: "Read the parked context under a limit that fits what assemble may write, or refuse at assemble with the size named, and state whichever limit remains in the chapter's Disclosed limits."
---

The scribe chapter says ingest proves the parked context and then runs its checks (.abcd/development/brief/04-surfaces/32-scribe.md:79), and its Disclosed limits name no size limit. `scribe ingest` reads the context under reading.MaxFileBytes, 4 MiB (internal/core/scribe/ingest.go:408), while `scribe assemble` writes it with no cap (internal/core/scribe/assemble.go:392). Over this repository's ledger of 4,797 records assemble exits 0 and parks a context of about 9.46 MB, and every ingest of it, --dry-run included, exits 2 with 'reading the context: fsutil: file exceeds size cap'. The scribe cannot land a record in a repository with a ledger this size, and the refusal comes only after a session has transcribed against that context.
