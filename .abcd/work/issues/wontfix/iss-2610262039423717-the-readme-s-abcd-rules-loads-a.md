---
schema_version: 1
id: "iss-2610262039423717"
slug: "the-readme-s-abcd-rules-loads-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 880 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:283"
remedy: "none (filed automatically)"
related_intents: [itd-3]
finding_ref: cfk-b3d22b75bf44cabe
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038011053. Same itd-3 out-of-scope line against a different document that reads the machine layer."
duplicates: [iss-2610262038011053]
---

The README's `abcd rules` loads a machine-wide rules.json layer, which itd-3 explicitly rejects.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:283` — "the bundled default domains, then the machine's `~/.abcd.noindex/rules.json`, then the"
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:41` — "Keeping a personal cross-repo rules file recreates exactly the"

The rules verb section says the loader reads the bundled domains, then the machine's `~/.abcd.noindex/rules.json`, then the checkout's `.abcd/rules.json`. itd-3, the shipped intent the README names as delivering `rules`, lists a global user rules file as explicitly rejected, because it recreates the scaffolding-outside-the-repo failure, and says the loader is per-repo only. itd-3 does not record the later reversal, so the two documents give opposite answers on whether a user layer exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 880.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038011053. Same itd-3 out-of-scope line against a different document that reads the machine layer.
