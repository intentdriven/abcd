---
schema_version: 1
id: "iss-2610262037438474"
slug: "itd-50-calls-the-fidelity-verdict-s"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 744 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-50]
finding_ref: cfk-12e860c2c72f5307
---

itd-50 calls the fidelity verdict's agent intent-fidelity-reviewer; 34-build names the same validator intent-auditor

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25` — "the `intent-fidelity-reviewer` records a `NOT_MET` verdict"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:684` — "`intent-auditor`. The fidelity audit runs once, on that lane"

itd-50's press release says the intent-fidelity-reviewer records the NOT_MET verdict that its loop acts on. 34-build gives that role on the closing lane, which writes the per-criterion fidelity verdict, to an agent named intent-auditor. These are two names for one agent, and a builder wiring itd-50's loop would look for a definition that the build loop does not dispatch.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 744.
