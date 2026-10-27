---
schema_version: 1
id: "iss-2610262034363298"
slug: "itd-7-has-disembark-write-the-repoprompt"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 509 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-7-rp-workspace-portability.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-7]
finding_ref: cfk-0e026b5f6d912562
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031532231. Same End A (itd-7 line 20, disembark writes into the repo); the board chapter states disembark writes only to an operator-chosen destination."
duplicates: [iss-2610262031532231]
---

itd-7 has disembark write the RepoPrompt workspace into the repository; the board chapter says disembark is read-only and writes only to an operator-chosen destination.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-7-rp-workspace-portability.md:20` — "When `dev-sync` runs (and as part of `/abcd:disembark`), abcd reads the project's RepoPrompt workspace definition from `~/Library/Application Support/RepoPrompt/Workspaces/` and writes it into `.abcd/rp/workspace.json` in the repo"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:573` — "because disembark is read-only and writes to an operator-chosen destination"

itd-7 says that as part of /abcd:disembark abcd reads the RepoPrompt workspace definition and writes it into .abcd/rp/workspace.json in the repo. 08-abcd states that disembark is read-only and writes to an operator-chosen destination, which is why there is no in-tree lifeboat directory for the board to read. A disembark that writes a file into the source repository is not read-only.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 509.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031532231. Same End A (itd-7 line 20, disembark writes into the repo); the board chapter states disembark writes only to an operator-chosen destination.
