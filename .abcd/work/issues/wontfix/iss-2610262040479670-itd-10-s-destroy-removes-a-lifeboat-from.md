---
schema_version: 1
id: "iss-2610262040479670"
slug: "itd-10-s-destroy-removes-a-lifeboat-from"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 955 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-10-purge-uninstall.md:16"
remedy: "none (filed automatically)"
related_intents: [itd-10]
finding_ref: cfk-68dc51f3f7398fc7
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031329436. Identical A end and the same out-of-tree lifeboat fact, cited from the disembark glossary."
duplicates: [iss-2610262031329436]
---

itd-10's destroy removes a lifeboat from inside .abcd/; the disembark glossary says there is no in-tree lifeboat directory.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-10-purge-uninstall.md:16` — "AND the entire `.abcd/` namespace (including lifeboat, dev artefacts, logbook, and config)"
- End B: `.abcd/development/brief/glossary/core/disembark.md:24` — "so there is no in-tree lifeboat directory"

itd-10's destroy deletes the .abcd/ namespace and lists the lifeboat among its contents. The disembark entry says the source tree is never written to, so no in-tree lifeboat directory exists; the lifeboat lands at a destination outside the repository. Destroy cannot remove a lifeboat that is never in the namespace.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 955.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031329436. Identical A end and the same out-of-tree lifeboat fact, cited from the disembark glossary.
