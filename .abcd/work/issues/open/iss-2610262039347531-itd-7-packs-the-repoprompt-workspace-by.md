---
schema_version: 1
id: "iss-2610262039347531"
slug: "itd-7-packs-the-repoprompt-workspace-by"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 870 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-7-rp-workspace-portability.md:38"
remedy: "none (filed automatically)"
related_intents: [itd-7]
finding_ref: cfk-a2aa10111ea0614c
---

itd-7 packs the RepoPrompt workspace by running dev-sync, which writes into the repo, as a disembark pre-pack step 'already in brief'; the configuration chapter says disembark never writes to the source repository

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-7-rp-workspace-portability.md:38` — "dev-sync runs as a pre-pack step (already in brief)"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:505` — "`disembark` never writes to the source repository."

itd-7 relies on dev-sync running as a pre-pack step of /abcd:disembark and writing `.abcd/rp/workspace.json` into the repository, and it says the step is already in the brief. The configuration chapter makes it load-bearing that disembark never writes to the source repository. It stages dev-sync in full, and it records as an unsettled open question whether a writing sweep can be part of disembark at all. Both cannot hold: the intent's pack mechanism needs a write that the brief rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 870.
