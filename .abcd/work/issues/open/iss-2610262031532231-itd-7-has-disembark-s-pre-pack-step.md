---
schema_version: 1
id: "iss-2610262031532231"
slug: "itd-7-has-disembark-s-pre-pack-step"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 190 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-7-rp-workspace-portability.md:20"
remedy: "Rewrite itd-7's press release, Pack bullet and criteria so disembark copies an existing .abcd/rp/workspace.json into the lifeboat destination and never writes the source repository; keep the dev-sync pull as a separate step."
related_intents: [itd-7]
finding_ref: cfk-80b461b8a8f1d36d
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

itd-7 has disembark's pre-pack step write .abcd/rp/workspace.json into the repository being read; invariant 6 says disembark never writes to the source.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-7-rp-workspace-portability.md:20` — "writes it into `.abcd/rp/workspace.json` in the repo"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:19` — "Lifeboat is always *output*, and disembark never writes to the source"

itd-7 says that when dev-sync runs, and as part of /abcd:disembark, abcd reads the RepoPrompt workspace and 'writes it into `.abcd/rp/workspace.json` in the repo', with dev-sync run as a pre-pack step so the file reaches the lifeboat. Invariant 6 states 'Lifeboat is always *output*, and disembark never writes to the source', with the lifeboat written out-of-tree to an operator-chosen destination. A pack path that writes into the source repository breaks that invariant, so the planned intent cannot be built as written unless the write leaves the pack path or the invariant is amended.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 190.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
