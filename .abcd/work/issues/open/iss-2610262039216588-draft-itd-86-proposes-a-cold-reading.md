---
schema_version: 1
id: "iss-2610262039216588"
slug: "draft-itd-86-proposes-a-cold-reading"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 855 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-86-cold-reading-surface.md:29"
remedy: "Rewrite itd-86 so it no longer proposes a one-document surface the agent catalogue already ships as four position prompts over the assembled input (reading assemble and reading ingest); keep only the residual scope (settled-enough check, closing line), or supersede it by itd-184 and move it to intents/superseded/."
related_intents: [itd-86]
finding_ref: cfk-8268a24ee9762f2b
---

Draft itd-86 proposes a cold-reading surface that the agent catalogue says already ships as the four-position instrument

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-86-cold-reading-surface.md:29` — "A surface that takes exactly one target document and produces candidate detections against the constraints visible in that document alone."
- End B: `.abcd/development/brief/05-internals/01-agents.md:99` — "The four `cold-reading-*` prompts are the reading positions of the cold-reading"

itd-86 is still a draft for a cold reading that takes exactly one document and reports each tension with the constraint in play. The agent catalogue says the cold-reading instrument ships as four position prompts (itd-184 / spc-62), dispatched over the input `abcd reading assemble` builds, and one of those positions returns exactly that detection shape. The two documents cover the same surface, and they disagree on what it reads: one design document, or an assembled slice of the shipped repository.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 855.
