---
schema_version: 1
id: "iss-2610262034153896"
slug: "draft-itd-169-promises-a-stopping-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 481 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-169-an-agent-loop-that-stops-says-who-it-is.md:12"
remedy: "none (filed automatically)"
related_intents: [itd-169]
finding_ref: cfk-6cbaaef51eba7011
---

draft itd-169 promises a stopping agent loop says whom it is asking; the board chapter's badge already has the agent name whom it addresses at a stop

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-169-an-agent-loop-that-stops-says-who-it-is.md:12` — "An agent loop that stops says who it is asking and what it needs, and never simply halts"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:182` — "state itself with `abcd mode`, naming whom it addresses."

itd-169 claims, as a draft, that an agent loop that stops says who it is asking and what it needs. The board chapter says that at a stop that is not a question the agent sets the badge state itself with abcd mode, naming whom it addresses, and that a chipped question sets it from the chip's role. The core of the draft's ground is already covered by the shipped badge.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 481.
