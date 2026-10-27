---
schema_version: 1
id: "iss-2610262038577903"
slug: "itd-25-calls-issue-scout-an-existing"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 829 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:53"
remedy: "none (filed automatically)"
related_intents: [itd-25]
finding_ref: cfk-eab9560c26b2eeab
---

itd-25 calls issue-scout an existing agent; 01-agents lists it in a design roster of which none exists

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:53` — "The existing `issue-scout` agent already covers GitHub upstream-issue annotation"
- End B: `.abcd/development/brief/05-internals/01-agents.md:80` — "none of them exists in `agents/`:"

itd-25 sets its out-of-scope boundary on 'the existing issue-scout agent', which it says already covers upstream-issue annotation. 01-agents puts issue-scout in the design-target roster and says none of those agents exists in agents/. The draft relies on an agent that has not been built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 829.
