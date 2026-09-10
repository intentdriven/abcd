---
schema_version: 1
id: "iss-2609100507421759"
slug: "nothing-tells-you-an-open-issue-is-already-fixed"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (capture, lint)"
---

Nothing tells you an open issue is already fixed. The ledger goes stale silently, and the cost of finding out lands on whoever plans the next piece of work.

Observed opening an autonomous sweep over a managed repository's ledger. Four open issues had been fixed on the default branch by merged pull requests whose commit subjects name the fix, but nobody ran `abcd capture resolve`, so the ledger still listed them as open. Sorting it out meant diffing the tree against each record by hand before any work could be assigned, roughly an hour before the sweep proper started. There is no marker anywhere that says "this was fixed"; the only evidence is in commit prose the ledger never reads.

This is the same failure mode the resolve-in-the-same-change convention exists to prevent, seen from the other side: the convention is a discipline, and a discipline that lapses leaves no trace. The tool holds both halves of the evidence — the record's id and the default branch's commit messages — and never puts them together.

Wanted: a lint (a `capture lint`, or a row in `abcd lint`) that flags an open issue whose id appears in a commit message on the default branch, or whose `found_at` file changed in a commit whose body cites the id, as "possibly resolved". `capture resolve --commit` already exists, so the lint could suggest the sha it found and the operator could accept it. False positives are cheap here — a mention is not a fix, and a human reads the row — while the current silence is not.
