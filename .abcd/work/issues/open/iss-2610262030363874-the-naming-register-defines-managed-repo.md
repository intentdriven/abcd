---
schema_version: 1
id: "iss-2610262030363874"
slug: "the-naming-register-defines-managed-repo"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 22 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:223"
remedy: "none (filed automatically)"
finding_ref: cfk-4142a6ea8b0943cc
---

The naming register defines `managed-repo` as needing `.git/` and accepting an in-tree `.abcd/`; the ahoy chapter rejects both

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:223` — "Folder kind: a git repo abcd already manages — has an ABCD marker block (or an in-tree `.abcd/`, or an `index.json` entry) and a `.git/` directory."
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:568` — "An in-tree `.abcd/` directory is recorded as a signal and reported, but it does not make a folder managed on its own (iss-88): a directory holding nothing but `.abcd/` reports as `unmanaged-folder`. A `.git/` directory means the folder is *a* repo, not that it is *managed*, and a folder carrying a marker block is treated as a managed repo whether or not it is a git checkout at all."

The reserved-vocabulary row defines `managed-repo` as a git repo with a marker block, an in-tree `.abcd/` or an index entry, and a `.git/` directory. The ahoy chapter, which the row cites as its source, says an in-tree `.abcd/` alone does not make a folder managed, and that a folder with a marker block is managed whether or not it is a git checkout. The register's definition disagrees with the classifier it points to.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 22.
