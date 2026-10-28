---
schema_version: 1
id: "iss-2610262033311711"
slug: "the-report-chapter-says-the-sources"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 415 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/07-library.md:109"
remedy: "none (filed automatically)"
finding_ref: cfk-f1519bb6179d5ad8
---

The report chapter says the sources store is keyed by the full root-commit SHA; the library chapter names a repository's sources ledger by the first twelve hex digits of its root commit.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/07-library.md:109` — "A repository's ledger is named by the first twelve hex digits of its root commit"
- End B: `.abcd/development/brief/04-surfaces/29-report.md:95` — "The inbox is a machine-scoped store beside the history, transcript, worktree, sources, labs and run stores. The sender key is the reporting repository's full root-commit SHA, the key those stores use"

29-report lists the sources store among the machine-scoped stores and says the full root-commit SHA is 'the key those stores use'. 07-library says a repository's ledger in the sources corpus is named by the first twelve hex digits of its root commit. The two pages cannot both describe the sources store's key correctly.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 415.
