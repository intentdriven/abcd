---
schema_version: 1
id: "iss-2610151422061013"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/10-docs.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/10-docs.md:121-123 to match the shipped surface: Only the citation leaves take them: `abcd docs cite confirm` and `abcd docs cite refresh` carry --config and --root. The other working verbs under `abcd docs`, `fidelity` (flags --apply, --autonomous, --intent, --report) and `fidelity record` (flag…"
---

The brief at .abcd/development/brief/04-surfaces/10-docs.md:121-123 says: "The working verbs accept the `docs-lint.json` to load and the repo to work over; the bare citation parent only routes and takes neither." The shipped surface differs: Only the citation leaves take them: `abcd docs cite confirm` and `abcd docs cite refresh` carry --config and --root. The other working verbs under `abcd docs`, `fidelity` (flags --apply, --autonomous, --intent, --report) and `fidelity record` (flag --verdict-json), take neither, as the chapter's own generated appendix (lines 273-290) shows. The sentence holds for the cite sub-tree but is stated of the docs verb's working verbs generally. (v0.14.2 release crosscheck, false-claim, raised by opus in a09.)
