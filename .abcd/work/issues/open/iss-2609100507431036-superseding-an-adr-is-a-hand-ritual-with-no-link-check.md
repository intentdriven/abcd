---
schema_version: 1
id: "iss-2609100507431036"
slug: "superseding-an-adr-is-a-hand-ritual-with-no-link-check"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (decide, record-lint)"
deferred_after: "v0.11.1"
deferral_reason: "planning F owed to the product thinker: partly present. record_schema already refuses a one-way supersession in either direction (internal/core/lint/schema.go, since 2026-08-02), but only in this repository's record-lint, not in abcd lint for a managed repository (the fold iss-45 asks about). Still unbuilt: abcd decide --supersedes (decide takes no flags) and a check that every ADR has a row in the ADRs README (index_drift has no ADR index). (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
remedy: "Waits on planning F (decide --supersedes and an ADR index row): if planned, add abcd decide --supersedes adr-N that mints the new ADR with supersedes, writes the old ADR's status and superseded_by and both README rows in one write, and extend index_drift to fail an ADR file with no ADRs README row, proven by a decide test asserting all five edits and an index_drift test with a missing row; the two-ended agreement check already exists in record_schema (internal/core/lint/schema.go) and reaches managed repositories only with the fold iss-45 asks about; if declined, wontfix keeping that check."
---

Superseding an ADR is a five-file hand ritual and nothing checks that the two ends of the link agree.

Observed in a managed repository whose stated rule is "never change an ADR, supersede it". Doing that meant, per supersession: a new file, `supersedes:` in its frontmatter, a Supersedes line in its body, the old file's `status` and `superseded_by` fields, and an index README row for both. Three ADRs were superseded this way in one day. Nothing verifies that a `superseded_by` has a matching `supersedes` in the named record, or the reverse, so a half-finished supersession reads as a complete one.

The same repository turned up the neighbouring gap in the same sweep: an ADR filed on one day had no row in the ADRs README until the next. The README says the row is a hand edit, and nothing checks it, so an ADR can exist and be invisible to every reader who starts at the index.

Wanted: `abcd decide --supersedes adr-N` doing all five edits as one operation; and a record-lint row, beside the existing record-provenance rule, that fails when an ADR file has no row in the ADRs README, and when `superseded_by` and `supersedes` disagree in either direction. Both are the same class of check the ledger already applies to issues by folder membership; ADRs get it by hand or not at all.

## Remedy grounds (2026-09-29)

Why: the ritual is five mechanical edits and the one-way check already exists, so the gap is the writer and the index row. SOTA check: adr-tools supersedes in one command, adr new -s N, which marks the old record superseded and links the new one (https://github.com/npryce/adr-tools, read 2026-09-29), and MADR records it in status as superseded by ADR-NNNN (https://adr.github.io/madr/, read 2026-09-29). Adopted the one-command shape, not the tool, which would add a shell toolchain beside the Go binary.
