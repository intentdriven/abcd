---
schema_version: 1
id: "iss-2610262032379645"
slug: "adapters-has-ahoy-redirecting-prompt"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 294 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/02-adapters.md:289"
remedy: "In 02-adapters.md reword the prompt-exports redirect to the future tense and either choose a home outside the registry-only history folder or say the ahoy inventory gains the folder when it is built."
finding_ref: cfk-c1ffd3841d8cae0d
---

adapters has ahoy redirecting prompt exports into ~/.abcd.noindex/history/< root-sha>/; ahoy says history/ holds the registry only

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/02-adapters.md:289` — "which ahoy redirects into `~/.abcd.noindex/history/<root-sha>/prompt-exports/` via the `<repo>/prompt-exports` symlink"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:427` — "the REGISTRY only: identity and lineage keyed on the"

The adapters chapter says ahoy redirects RepoPrompt prompt exports into a prompt-exports folder under ~/.abcd.noindex/history/< root-sha>/, through a symlink in the repository. The ahoy chapter defines history/ as the registry only, holding identity and lineage, and its inventory has no prompt-exports entry and no such symlink. Both cannot be true.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 294.
