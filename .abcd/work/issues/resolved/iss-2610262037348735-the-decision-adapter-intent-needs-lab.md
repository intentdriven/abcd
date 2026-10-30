---
schema_version: 1
id: "iss-2610262037348735"
slug: "the-decision-adapter-intent-needs-lab"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 733 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md:45"
remedy: "In the decision-adapter intent state that lab harvest gains a per-type agreement section computed from the adapter's recorded pairs (new scope on the shipped lab verb), or name a new sub-verb for it; the lab chapter describes a harvest that only assembles prose sections."
related_intents: [itd-2609221009495079]
finding_ref: cfk-98716f3a82450774
resolution: "lab harvest gains a per-type agreement section computed from the decision adapter's pairs.jsonl (counts, compared, agreed, skipped, rate, four probability bands, and the research note's line naming the lab home and the count); the lab chapter now describes it"
impact: internal
resolved_by:
  intent: "itd-2609221009495079"
  spec: "spc-2609221011151661"
---

The decision-adapter intent needs lab harvest to report agreement rates with probability bands; the lab brief's harvest only assembles prose sections from findings and probe records

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md:45` — "`abcd lab harvest` produces agreement rates per type with counts and probability bands"
- End B: `.abcd/development/brief/04-surfaces/31-lab.md:99` — "assembles the lab's harvest in the lifeboat's section shape — intention, method, what worked, what is open, candidates, coverage — from the INTENTION, the findings log and the probe records"

The intent's evidence for turning a judgement type on is the agreement rates per type, with counts and probability bands, that `abcd lab harvest` produces from pairs the adapter recorded. In the lab chapter, the harvest assembles the lifeboat's sections from the INTENTION, the findings log and the probe records, and computes no statistics. The lab verbs run nothing either, because whoever runs the lab runs each probe. As the lab is written, its harvest cannot produce the figures this intent depends on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 733.

## Grounds

- pursued: the harvest's agreement section is the evidence path the intent names; shown wrong if a lab holding pairs harvests without per-type counts and bands
