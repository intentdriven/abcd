---
schema_version: 1
id: "iss-2610262037365749"
slug: "the-model-tier-intent-says-nothing-reads"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 735 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:26"
remedy: "none (filed automatically)"
related_intents: [itd-2609170822093401]
finding_ref: cfk-b45de80ed8e456d0
refines: [itd-2609170822093401]
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both statements were true when written; the setting the intent calls unread is still unread, and the setting the build chapter reads is another one."
---

The model-tier intent says nothing reads the oracle setting and every delegated step goes where the harness picks; the build brief reads oracle.review to route a lane's reviews

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:26` — "backend is one repo-wide value that nothing reads, and every delegated step"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:544` — "`oracle.review` in the layered configuration names `host`, the bundled default, or `rp`"

The intent's press release starts from the premise that the oracle backend is a single repo-wide value nothing reads, so every delegated step reaches whatever model the harness picks. The build chapter says `oracle.review` in the layered configuration sends a build lane's ruthless and security reviews to `rp` instead of the host. The setting is read, and not every delegated step goes to the harness's choice.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 735.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both statements were true when written; the setting the intent calls unread is still unread, and the setting the build chapter reads is another one.
