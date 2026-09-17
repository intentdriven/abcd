---
schema_version: 1
id: "iss-2609100508570527"
slug: "capture-does-not-say-that-the-record-it-wrote-is-untracked"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (capture, status render)"
---

`abcd capture` writes the record file and never says the record is not in git, so a ledger entry can be invisible to every branch and every gate that reads the committed tree.

Observed during an autonomous run in a managed repository. Six issue records existed only as untracked files in the primary checkout. A worker branched from the default branch could not see them, could not resolve them, and had no signal that they existed at all; the orchestrator found the gap by listing the directory rather than by any tool output. `abcd capture` reported success on each, and the bare status render listed them alongside committed records with nothing to distinguish the two.

The store's status model is folder membership, and folder membership is only a status signal once the file is committed. An uncommitted record is in no state at all: it is not open to anyone but the checkout that holds it.

Wanted: have `abcd capture` say, at write time, that the record it just wrote is untracked and needs committing; and have the status render mark an untracked or uncommitted record as such rather than showing it as an equal member of its folder. Both are a `git status` read the tool can already do — this repository's own conventions treat an uncommitted peer diff as significant, and the ledger's own writes are the one place that signal is currently dropped.
