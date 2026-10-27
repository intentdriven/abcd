---
schema_version: 1
id: "iss-2610262041227792"
slug: "itd-7-has-disembark-write-the-repoprompt"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 995 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-7-rp-workspace-portability.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-7]
finding_ref: cfk-d0553f2fc1b7d3bd
---

itd-7 has disembark write the RepoPrompt workspace file into the repository; the glossary defines disembark as reading the source repository without writing to it

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-7-rp-workspace-portability.md:20` — "When `dev-sync` runs (and as part of `/abcd:disembark`), abcd reads the project's RepoPrompt workspace definition from `~/Library/Application Support/RepoPrompt/Workspaces/` and writes it into `.abcd/rp/workspace.json` in the repo."
- End B: `.abcd/development/brief/glossary/core/disembark.md:4` — "reads a source repository without writing to it"

itd-7 has abcd write `.abcd/rp/workspace.json` into the repository "as part of `/abcd:disembark`", with dev-sync run as a pre-pack step. The canonical glossary entry defines disembark as packing that reads a source repository without writing to it, which is the read-only guarantee the lifeboat surface rests on. A disembark run cannot both write a file into the source tree and leave it unwritten, so itd-7 cannot be built as written without breaking that guarantee.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 995.
