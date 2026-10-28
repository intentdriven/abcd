---
schema_version: 1
id: "iss-2610262037281449"
slug: "the-sessions-announce-intent-plans-to"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 725 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120"
remedy: "none (filed automatically)"
related_intents: [itd-2609151838312703]
finding_ref: cfk-f9d68b54a6967953
---

The sessions-announce intent plans to rename the status board (bare abcd) to the dashboard; the dashboard brief already uses that name for the Tailscale web server

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120` — "that change and renaming the status board to the dashboard are captured as their own records"
- End B: `.abcd/development/brief/04-surfaces/36-dashboard.md:3` — "`/abcd:dashboard` runs the one network listener abcd has: a server on this computer that the product thinker opens from a phone"

Decision 21 of the intent records that the status board, which bare `abcd` renders, is to be renamed the dashboard. The dashboard chapter gives `/abcd:dashboard` to something else: abcd's one network listener, a server on this computer that the product thinker opens from another device. If the rename goes ahead, one name will cover two different surfaces.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 725.
