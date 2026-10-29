---
schema_version: 1
id: "iss-2610262032458879"
slug: "the-verification-matrix-checks-that-ahoy"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 312 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:1070"
remedy: "In 06-delivery/02-verification-matrix.md qualify the 'ahoy rules-loader install' row: the marker block is injected only where docs.target is agents_md (a default install writes none), while the rules override skeleton and the hook-manifest check are what a default install does."
finding_ref: cfk-36af3cf1f3c88b6d
---

the verification matrix checks that ahoy's rules-loader install injects the marker block; ahoy's default docs target writes the block into no conventions file

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:1070` — "a default install writes it into none of the repository's committed conventions files"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:30` — "Marker block from the bundled defaults injected; the rules override skeleton written"

02-verification-matrix's ahoy rules-loader install row, unstaged and so a claim about shipped behaviour, asserts the marker block from the bundled defaults is injected. 01-ahoy says the docs target defaults to skip, so a default install writes the block into none of the repository's committed conventions files and only an agents_md choice plants it. A default-install test written from the matrix row would fail against the behaviour ahoy specifies.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 312.
