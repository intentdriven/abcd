---
schema_version: 1
id: "iss-2610071538088019"
slug: "route-work-about-abcd-itself-to-abcd-report-never-into-a"
severity: "minor"
category: "process"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071224565746 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd intent, abcd capture, intent decomposition, bundled rule domains"
remedy: "none (filed automatically)"
---

Route work about abcd itself to abcd report, never into a managed repository's intents or issues

## What happened

In a managed repository (pge, a game engine), an accepted decision about how pge settles borderline banned names called for three new capabilities: in-session judging of warn-tier hits, a local-model shadow log, and a review of that log. All three are features of abcd's name check, not of pge. They were nevertheless filed as three pge draft intents (commit 490d14d). The decomposition step on 2026-10-03 did offer abcd as their home; the owner declined then, and nothing restated the question.

On 2026-10-07 the drafts reached a planning interview in pge before the owner caught it ("are these not abcd questions? Why in pge?"). They were then sent to abcd as rpt-2610071222173136 and removed from pge (commit 79e9eba). Planning time had already been spent on a record in the wrong repository.

## The owner's convention

In the owner's words: "anything related to abcd (the tooling) should go in via /abcd:report", and this "must become an intent for abcd to implement".

The boundary: what the tool does goes to abcd through a report; what the managed repository decides about its own use of the tool (its policy, as in adr2610031742534388) stays in that repository and cites the report.

## What abcd would change

1. **Teach it.** A bundled rule, recalled on intent, capture and decomposition work, states the convention and the boundary above. Today the report page says findings about the working repository go to capture, but nothing says the converse at the moment an intent or issue is filed.
2. **Route it in decomposition.** When a part of a decomposition is a change to abcd's own verbs, hooks, checks or pages, its default home is a report, and a decision to keep it in the managed repository is recorded with its reason.
3. **Check it at filing.** abcd intent and abcd capture warn, without refusing, when the text names abcd's own surfaces (for example the name check, the banlist, drain, the status line), and name the report verb as the route.

## How to see it again

Decompose a managed repository's decision that requires a new abcd capability, accept the default homes, and file the parts: the capability lands as the managed repository's own intent with no prompt to report it.

Remedy the reporter proposes: Make it abcd's convention, taught and checked: a bundled rule recalled on intent and capture work says abcd-tooling work goes to abcd report; decomposition routes such parts to a report by default; intent and capture warn when the text is about abcd's own surfaces.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610071224565746, a enhancement against abcd v0.13.1, surface abcd intent, abcd capture, intent decomposition, bundled rule domains.

Evidence:

- rpt-2610071224565746 (the report, kept in the inbox)
- rpt-2610071222173136
- adr2610031742534388
- 490d14d
- 79e9eba
- lab-261002171217-c372ff6

Every record id the report names is its sender's own, not this repository's, so each is written as one word, family and number together, and cites nothing here.
