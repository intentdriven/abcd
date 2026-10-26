---
schema_version: 1
id: "iss-2610262031537827"
slug: "itd-7-depends-on-a-dev-sync-pre-pack"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 191 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-7-rp-workspace-portability.md:38"
remedy: "none (filed automatically)"
related_intents: [itd-7]
finding_ref: cfk-c37daaf29ad4506a
---

itd-7 depends on a dev-sync pre-pack step it calls 'already in brief'; the brief lists dev-sync only among out-of-scope work not yet under way.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-7-rp-workspace-portability.md:38` — "dev-sync runs as a pre-pack step (already in brief)"
- End B: `.abcd/development/brief/01-product/01-press-release.md:33` — "scheduled `dev-sync`, opencode harness portability"

itd-7 hangs both its pull and its pack on dev-sync: the workspace is written on dev-sync, and 'dev-sync runs as a pre-pack step (already in brief)'. The brief's press release lists 'scheduled `dev-sync`' among the out-of-scope items that are not yet under way and live as draft intents, and describes the pack as three passes with no dev-sync step. As written, the planned intent depends on a verb and a pack step the brief records as unbuilt, and its claim that the step is already in the brief does not hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 191.
