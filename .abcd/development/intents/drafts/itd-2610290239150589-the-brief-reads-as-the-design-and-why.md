---
id: itd-2610290239150589
slug: the-brief-reads-as-the-design-and-why
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
related_adrs: [adr-2610290238168123]
severity: major
origin: researcher-authored
production_mode: hand-written
---

# The brief reads as the design and why, and each command's behaviour is written once, in the page agents follow

## Press Release

> **abcd's brief now reads as the product being built and the reasons for it, and every command's behaviour is written once, in the command page an agent follows when it runs the command.** A product thinker opening a command's chapter finds why the command exists, the trade-offs taken and why it refuses what it refuses, with a pointer to the page for exactly what it does. The page is the copy the documentation check now reads against the code, so the copy that does harm when it is wrong is the one that cannot drift unnoticed. The brief shrinks by about two-thirds, and a second hand-kept copy no longer exists to contradict the first.
>
> "I used to find the same rule written three ways and no way to tell which one the agents were following," said a product thinker who read the brief after a consistency pass filed thirteen hundred conflicts. "Now the brief tells me what we are building and why, and when I want to know exactly what a command does, it sends me to the one place that says so."

## Why This Matters

The brief-size study of 2026-10-25 measured the brief at about 247,000 words, about 69% of it restating another file; the command chapters alone were 76% restatement (about 121,000 words), and the paraphrased copies had drifted apart in both directions. The 2026-10-26 consistency pass filed 1,301 records, and 1,080 of its 1,307 findings had a passage of the brief at one end; 45 major ones are deferred until this rewrite lands. The product thinker ruled on 2026-10-27 that everything has one home (dcn-2610271112331508), that behaviour lives in the command page (dcn-2610271112339516) and that the brief states the target design (dcn-2610271112330235); the ADR that records the reversal is adr-2610290238168123.

## Mechanism

We expect moving each behaviour claim to its command page, and checking the pages against the code instead of the brief, to remove the class of contradiction the pass found, because a claim with one home has nothing to disagree with.

## What's In Scope

- **The command chapters**: each chapter under `04-surfaces/` keeps why the command exists, its trade-offs, why it refuses what it refuses, its `## Sub-verbs` table and its generated appendix; each behaviour passage moves to the command page under `commands/`, or is cut where the page already says it, keeping any sentence only the chapter held.
- **The documentation-fidelity check** reads the command pages for behaviour claims, its review instructions and receipts with it.
- **The brief's other copies**, as the one-home ruling settles: invariants become one line naming the rule and its ADR; the verification matrix is generated from the tests or removed; the out-of-scope list points to the drafts shelf; one machine-folder inventory; superseded designs still staged are deleted.
- **The status overlay**: the brief states the target design in the present tense, and a built part carries a tag saying so; the truth rule in `00-meta.md` is amended to that one variance.
- **The deferred consistency majors**: each of the 45 is resolved by the change that rewrites its passage, or closed if the passage is gone.

## What's Out of Scope

- The glossary, which stays the one home for terms (its 17 stale entries are fixed under their own records).
- The generated appendices and the shape rule, which are unchanged.
- `docs/`, the user-facing documentation, which has its own currency review.

## Decisions

1. **One home per claim** (dcn-2610271112331508).
2. **The command page is the home of behaviour** (dcn-2610271112339516).
3. **The brief is the target design with a status overlay** (dcn-2610271112330235).

## Acceptance Criteria

- **Given** a command chapter under `04-surfaces/`, **when** the rewrite lands, **then** it states no refusal, output, step or exit code that its command page does not hold, and it points to the page for behaviour.
- **Given** a behaviour claim in a command page that the code contradicts, **when** the documentation-fidelity review runs, **then** the claim is a finding that holds a spec close and a release cut, as a false brief sentence does today.
- **Given** the brief after the rewrite, **when** its words are counted, **then** it holds about a third of today's, and every section the brief-size study marked as restating another file is a pointer.
- **Given** each of the 45 deferred consistency majors, **when** the rewrite lands, **then** it is resolved by the change that rewrote its passage, or closed because the passage is gone.
