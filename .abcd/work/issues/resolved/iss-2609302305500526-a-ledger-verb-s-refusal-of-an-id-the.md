---
schema_version: 1
id: "iss-2609302305500526"
slug: "a-ledger-verb-s-refusal-of-an-id-the"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "v0.12.0 release gate crosscheck (autonomous run A)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/alloc.go"
remedy: "Check the id against the ledger before the allocation lock is taken, or remove the lock and every directory the lock step created on a refusal; a test in a bare git repository runs each refusing verb and asserts git status --porcelain is empty afterwards."
resolution: "Every ledger verb that names an existing record (resolve, wontfix, link, defer, remedy, promote on both routes, disposition, admit) now runs its read-only pre-flight before mutationPreamble, so a refusal of an id the ledger does not hold writes nothing; TestRefusalOfAnUnheldIDWritesNothing runs each verb in an empty root."
impact: fix
resolved_by:
  commit: "81395fdca"
---

A ledger verb's refusal of an id the ledger does not hold writes to the tree: in a git checkout with no ledger, capture resolve, wontfix, link, defer, promote and remedy over an id the ledger does not hold each exit 2 as the capture chapter promises (06-capture.md:294-300, 'nothing written') but leave .abcd/work/issues/.iss-alloc.lock behind, creating the directory chain (lock name internal/core/capture/alloc.go:19). abcd's own .gitignore hides it here; in any other repository it shows as an untracked file. Refusals caught before the lock is taken (a missing --impact, a lone word) leave the tree empty. Found by the v0.12.0 release-gate brief-surface cross-check (x-025, checker a05), reproduced by the classifier at b89784c4; standing since before v0.11.1.

## Grounds

- pursued: a refusal of an unheld id leaves the tree exactly as it was; a verb added later that calls the preamble before its lookup, or a status line after any refusing verb in a bare repository, would show it wrong
