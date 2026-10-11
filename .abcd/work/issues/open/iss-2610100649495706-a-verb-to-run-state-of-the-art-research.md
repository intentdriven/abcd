---
schema_version: 1
id: "iss-2610100649495706"
slug: "a-verb-to-run-state-of-the-art-research"
severity: "minor"
category: "future-work-seed"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071608595198 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd ideate, sota-researcher agent, abcd ingest and consult, research notes"
remedy: "none (filed automatically)"
---

A verb to run state-of-the-art research and keep it as a research note in the record

## What is missing

The owner of a managed repository asked whether abcd has a verb to conduct state-of-the-art research and store it as a research note, or whether that goes through an intent. abcd has neither:

- **ideate** runs primary-source research inside its admission gauntlet, but only to admit or reject one idea; its record is a verdict, not a survey of a field.
- **The sota-researcher agent** does the research and returns ranked recommendations with sources, but it is read-only, so nothing keeps its output. Saving it is the session's hand work.
- **ingest and consult** keep sources and record which sources fed which decisions; neither writes the note.
- **An intent is the wrong home:** research is input to decisions, not a capability anyone uses.

In practice the research note is written by hand (a lab baseline note was produced this way, through the agent). Nothing checks its shape, dates it, or lets an ADR or intent cite it as a record.

## What the verb would do

1. Take a question (and optionally the record it serves: an ADR, an intent, an open question).
2. The host runs the sota-researcher on it, outside the loop of whoever asked.
3. The verb validates the returned note against a declared shape: the question, ranked findings, each with an evidence tier and primary sources, and what would change the conclusion.
4. It writes the note dated under the development record's research notes, stamps an id, and records the link both ways, so a decision citing it is checkable.

## Why now

The owner's vision-refinement work in that repository keeps hitting questions where a state-of-the-art check is the presumptive input (abcd's own "prefer SOTA" principle), and each one currently ends as an unvalidated file or not at all.

Remedy the reporter proposes: Add a research verb: the host runs the sota-researcher on a question, the verb validates the returned note (question, ranked findings, evidence tiers, primary sources) and writes it dated under the development record's research notes, with a link field decisions and intents can cite.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610071608595198, a enhancement against abcd v0.13.1, surface abcd ideate, sota-researcher agent, abcd ingest and consult, research notes.

Evidence:

- rpt-2610071608595198 (the report, kept in the inbox)
- lab-261002171217-c372ff6
- rpt-2610071222173136
