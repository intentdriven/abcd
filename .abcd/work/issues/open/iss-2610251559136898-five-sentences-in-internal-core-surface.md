---
schema_version: 1
id: "iss-2610251559136898"
slug: "five-sentences-in-internal-core-surface"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D4"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/sentences.go"
remedy: "Reword the five write clauses within the 160-character SentenceCap, for example 'Remove abcd here, leaving .abcd/: Writes the removal of the marker block, README parts and PATH copy, and restores the status line; refuses any argument.', 'Writes the moved record and links to it', 'Writes the run's state, its reservation and the reason as the lane's first commit; refuses when none passes' and 'Writes drafted edits only with --apply or --autonomous'; then regenerate the pages."
---

Five sentences in internal/core/surface/sentences.go name less than the verb writes; each renders on the verb's help and on docs/reference/cli/commands.md. (1) ahoy uninstall (sentences.go:47; page 158) says 'Writes the removal of the marker block, PATH copy, and provenance record'. Uninstall also removes the README parts (internal/core/ahoy/apply.go:2196, readme_apply.go:454) and restores the harness's status-line command (apply.go:2201, statusline_apply.go:280-310). (2) capture resolve and (3) capture wontfix (sentences.go:88, 92; page 672, 714) say 'Writes the moved record'. The transition also repoints every relative markdown link in the working tree that named the old path (internal/core/capture/workflow.go:715; internal/core/relink/relink.go:1-4). (4) build next (sentences.go:63; page 411) leaves out the reservation its start pushes to origin (internal/core/implement/loop/loop.go:412), which build's own sentence names. (5) docs fidelity (sentences.go:125; page 1141) says 'Writes drafted edits only with --apply', yet --autonomous applies them too (internal/surface/cli/docfidelity.go:128). Same class as iss-2610160608444348, which holds the drain sentence.
