---
schema_version: 1
id: "iss-2610262036512603"
slug: "itd-93-says-abcd-s-release-workflows-are"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 678 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/10-docs.md:131"
remedy: "none (filed automatically)"
related_intents: [itd-93]
finding_ref: cfk-e25cb0993b6b705e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032317368. Same End A and same resolution; itd-93's 'exact machinery a managed repo receives' overstates slightly because abcd's substitution set adds gates, but it is a shipped intent and needs no edit."
duplicates: [iss-2610262032317368]
---

itd-93 says abcd's release workflows are generated from the one scaffold template, so abcd runs exactly what a managed repo gets; the docs chapter says abcd's release workflow has a docs-currency step that the scaffolded template never names

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/10-docs.md:131` — "The release workflow's docs-currency step runs the lint in its release-gate mode; CI's step and the `docs-lint` make target each run the lint in its plain mode, the scaffolded release template names the verb nowhere"
- End B: `.abcd/development/intents/shipped/itd-93-abcd-scaffolds-a-hardened-changelog.md:79` — "The proven pattern and the shipped template are one artifact by construction, so every abcd release exercises the exact machinery a managed repo receives."

itd-93 says abcd's own release.yml and auto-release.yml are regenerated from the single scaffold template, so every abcd release runs 'the exact machinery a managed repo receives'. The docs chapter says abcd's release workflow has a docs-currency step that runs the lint in release-gate mode, while 'the scaffolded release template names the verb nowhere'. If the template lacks that step, abcd's release machinery is not the scaffolded one. If abcd's release machinery is the scaffolded one, the template would have to name the verb.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 678.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032317368. Same End A and same resolution; itd-93's 'exact machinery a managed repo receives' overstates slightly because abcd's substitution set adds gates, but it is a shipped intent and needs no edit.
