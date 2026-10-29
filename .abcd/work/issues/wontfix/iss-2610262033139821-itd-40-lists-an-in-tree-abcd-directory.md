---
schema_version: 1
id: "iss-2610262033139821"
slug: "itd-40-lists-an-in-tree-abcd-directory"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 376 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:569"
remedy: "none (filed automatically)"
related_intents: [itd-40]
finding_ref: cfk-0f76338cb0f7f9b6
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Ahoy cites iss-88 for the narrowing, so a later record already says which rule holds; the shipped intent states the earlier rule."
---

itd-40 lists an in-tree .abcd/ directory as a strong managed marker; the ahoy chapter says it does not make a folder managed on its own

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:569` — "recorded as a signal and reported, but it does not make a folder managed on its"
- End B: `.abcd/development/intents/shipped/itd-40-folder-classification.md:33` — "a CLAUDE.md/AGENTS.md marker block, an in-tree `.abcd/` directory, or an entry for `cwd`'s root-commit SHA"

The folder-classification intent puts an in-tree `.abcd/` directory among the strong markers that decide managed against unmanaged. The ahoy chapter says that directory is only recorded as a signal and a folder holding nothing but `.abcd/` reports as unmanaged-folder (iss-88). Under one rule a bare `.abcd/` folder classifies as managed-repo and under the other as unmanaged-folder, so a classifier cannot follow both.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 376.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Ahoy cites iss-88 for the narrowing, so a later record already says which rule holds; the shipped intent states the earlier rule.
