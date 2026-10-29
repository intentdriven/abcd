---
schema_version: 1
id: "iss-2610262045089996"
slug: "itd-7-has-disembark-run-dev-sync-which"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1254 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-7-rp-workspace-portability.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-7, itd-88]
finding_ref: cfk-7ea9a3ac55747eb8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031532231. Same End A (itd-7 line 20); itd-88 (shipped) rules out writing anything into the source repository."
duplicates: [iss-2610262031532231]
---

itd-7 has disembark run dev-sync, which writes .abcd/rp/workspace.json into the repo; itd-88 says disembark never writes into the source repository

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-7-rp-workspace-portability.md:20` — "When `dev-sync` runs (and as part of `/abcd:disembark`), abcd reads the project's RepoPrompt workspace definition from `~/Library/Application Support/RepoPrompt/Workspaces/` and writes it into `.abcd/rp/workspace.json` in the repo."
- End B: `.abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:48` — "Writing anything into the source repository, ever."

itd-88 delivers the disembark probe and packer and rules out writing anything into the source repository, ever. itd-7 makes disembark the trigger for dev-sync ('as part of `/abcd:disembark`'), and dev-sync writes `.abcd/rp/workspace.json` into the repository being packed. A disembark that writes a file into the source repo breaks itd-88's rule, so building itd-7 as written means changing one of the two.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1254.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031532231. Same End A (itd-7 line 20); itd-88 (shipped) rules out writing anything into the source repository.
